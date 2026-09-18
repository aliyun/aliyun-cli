package lib

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// JSON fields deliberately do not expose the SDK's Go/XML representation.
type machineListItem struct {
	Kind         string     `json:"kind"`
	Bucket       string     `json:"bucket"`
	Key          string     `json:"key,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	ETag         string     `json:"etag,omitempty"`
	StorageClass string     `json:"storage_class,omitempty"`
	Modified     *time.Time `json:"last_modified,omitempty"`
	Created      *time.Time `json:"created,omitempty"`
	Region       string     `json:"region,omitempty"`
	VersionID    string     `json:"version_id,omitempty"`
	Latest       *bool      `json:"is_latest,omitempty"`
	UploadID     string     `json:"upload_id,omitempty"`
}
type machineListResult struct {
	SchemaVersion string            `json:"schema_version"`
	Items         []machineListItem `json:"items"`
	Returned      int               `json:"returned"`
	Complete      bool              `json:"complete"`
	Truncated     bool              `json:"truncated"`
	NextCursor    string            `json:"next_cursor,omitempty"`
}

// Offset is within a replayed service page, never within the next page. PageSize
// is retained across resumes to keep the replay stable even if the limit changes.
type listCursor struct {
	Version   int    `json:"v"`
	Query     string `json:"q"`
	Stage     int    `json:"s"`
	Marker    string `json:"m,omitempty"`
	Secondary string `json:"n,omitempty"`
	Offset    int    `json:"o"`
	PageSize  int    `json:"p"`
	PageHash  string `json:"h,omitempty"`
}

func encodeListCursor(c listCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}
func decodeListCursor(s, query string, stages int) (listCursor, error) {
	c := listCursor{}
	if len(s) > 32768 {
		return c, fmt.Errorf("invalid OSS cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		err = json.Unmarshal(b, &c)
	}
	if err != nil || c.Version != 1 || c.Query != query || c.Stage < 0 || c.Stage >= stages || c.Offset < 0 || c.Offset > 1000 || c.PageSize < 1 || c.PageSize > 1000 {
		return c, fmt.Errorf("invalid OSS cursor or cursor belongs to a different list query")
	}
	return c, nil
}

func (lc *ListCommand) listMachine(m *machineInvocation) error {
	// Clear singleton state on every invocation.
	lc.payerOption = nil
	lc.filters = nil
	encoding, _ := GetString(OptionEncodingType, lc.command.options)
	target := CloudURL{}
	var err error
	if len(lc.command.args) > 0 {
		target, err = CloudURLFromString(lc.command.args[0], encoding)
		if err != nil {
			return err
		}
	}
	var valid bool
	valid, lc.filters = getFilter(os.Args)
	if !valid {
		return fmt.Errorf("--include or --exclude does not support format containing dir info")
	}
	payer, _ := GetString(OptionRequestPayer, lc.command.options)
	if payer != "" {
		if payer != string(oss.Requester) {
			return fmt.Errorf("invalid request payer: %s", payer)
		}
		lc.payerOption = oss.RequestPayer(oss.Requester)
	}
	directory, _ := GetBool(OptionDirectory, lc.command.options)
	versions, _ := GetBool(OptionAllversions, lc.command.options)
	stages := []string{}
	if target.bucket == "" {
		if err := lc.lbCheckArgOptions(); err != nil {
			return err
		}
		stages = append(stages, "buckets")
	} else {
		subject := lc.getSubjectType()
		if subject&objectType != 0 {
			if versions {
				stages = append(stages, "versions")
			} else {
				stages = append(stages, "objects")
			}
		}
		if subject&multipartType != 0 {
			stages = append(stages, "uploads")
		}
	}
	limit, _ := GetInt(OptionLimitedNum, lc.command.options)
	if limit == -1 {
		limit = 100
	}
	if limit < 1 || limit > 1000 {
		return fmt.Errorf("machine lists require --limited-num between 1 and 1000")
	}
	marker, _ := GetString(OptionMarker, lc.command.options)
	marker, err = lc.command.getRawMarker(marker)
	if err != nil {
		return err
	}
	secondary := ""
	secondaryOption := OptionVersionIdMarker
	if stages[0] == "uploads" {
		secondaryOption = OptionUploadIDMarker
	}
	secondary, _ = GetString(secondaryOption, lc.command.options)
	secondary, err = lc.command.getRawMarker(secondary)
	if err != nil {
		return err
	}
	// Bind the opaque token to the effective target, ordered filters, and initial
	// markers. Credentials are deliberately excluded from the token and its hash.
	endpoint, _ := GetString(OptionEndpoint, lc.command.options)
	uploadMarker, _ := GetString(OptionUploadIDMarker, lc.command.options)
	queryBytes, _ := json.Marshal(struct {
		Bucket, Prefix, Endpoint, Encoding, Payer, Marker, Secondary, UploadMarker string
		Directory                                                                  bool
		Stages                                                                     []string
		Filters                                                                    []string
	}{target.bucket, target.object, endpoint, encoding, payer, marker, secondary, uploadMarker, directory, stages, machineFilterIdentity(lc.filters)})
	hash := sha256.Sum256(queryBytes)
	query := hex.EncodeToString(hash[:])
	cursor := listCursor{Version: 1, Query: query, Marker: marker, Secondary: secondary, PageSize: int(limit)}
	if m.cursor != "" {
		cursor, err = decodeListCursor(m.cursor, query, len(stages))
		if err != nil {
			return err
		}
	}
	items, nextMarker, nextSecondary, truncated, err := lc.machinePage(target, stages[cursor.Stage], directory, cursor)
	if err != nil {
		return err
	}
	pageBytes, _ := json.Marshal(items)
	pageHash := sha256.Sum256(pageBytes)
	pageID := hex.EncodeToString(pageHash[:])
	if cursor.PageHash != "" && cursor.PageHash != pageID {
		return fmt.Errorf("OSS page changed while resuming; restart the listing")
	}
	if cursor.Offset > len(items) {
		return fmt.Errorf("OSS page changed while resuming; restart the listing")
	}
	result := machineListResult{SchemaVersion: "1", Items: []machineListItem{}}
	pos := cursor.Offset
	for pos < len(items) && len(result.Items) < int(limit) {
		item := items[pos]
		pos++
		key := item.Key
		if item.Kind == "prefix" {
			key = strings.TrimSuffix(key, "/")
		}
		if item.Kind != "bucket" && !doesSingleObjectMatchPatterns(key, lc.filters) {
			continue
		}
		result.Items = append(result.Items, item)
	}
	if pos < len(items) {
		cursor.Offset = pos
		cursor.PageHash = pageID
		result.NextCursor = encodeListCursor(cursor)
	} else if truncated {
		if nextMarker == cursor.Marker && nextSecondary == cursor.Secondary {
			return fmt.Errorf("OSS returned a non-advancing list cursor")
		}
		cursor.Marker = nextMarker
		cursor.Secondary = nextSecondary
		cursor.Offset = 0
		cursor.PageHash = ""
		result.NextCursor = encodeListCursor(cursor)
	} else if cursor.Stage+1 < len(stages) {
		cursor.Stage++
		cursor.Offset = 0
		cursor.PageHash = ""
		cursor.Marker = marker
		cursor.Secondary, _ = GetString(OptionUploadIDMarker, lc.command.options)
		cursor.Secondary, err = lc.command.getRawMarker(cursor.Secondary)
		if err != nil {
			return err
		}
		result.NextCursor = encodeListCursor(cursor)
	}
	result.Returned = len(result.Items)
	result.Truncated = result.NextCursor != ""
	result.Complete = !result.Truncated
	encoder := json.NewEncoder(m.writer)
	if m.format == "json" {
		return encoder.Encode(result)
	}
	for _, item := range result.Items {
		if err := encoder.Encode(struct {
			Type          string          `json:"type"`
			SchemaVersion string          `json:"schema_version"`
			Item          machineListItem `json:"item"`
		}{"item", "1", item}); err != nil {
			return err
		}
	}
	return encoder.Encode(struct {
		Type          string `json:"type"`
		SchemaVersion string `json:"schema_version"`
		Returned      int    `json:"returned"`
		Complete      bool   `json:"complete"`
		Truncated     bool   `json:"truncated"`
		NextCursor    string `json:"next_cursor,omitempty"`
	}{"summary", "1", result.Returned, result.Complete, result.Truncated, result.NextCursor})
}
func machineFilterIdentity(filters []filterOptionType) []string {
	result := []string{}
	for _, f := range filters {
		result = append(result, f.name, f.pattern)
	}
	return result
}

func (lc *ListCommand) machinePage(target CloudURL, stage string, directory bool, c listCursor) ([]machineListItem, string, string, bool, error) {
	items := []machineListItem{}
	if stage == "buckets" {
		client, err := lc.command.ossClient("")
		if err != nil {
			return nil, "", "", false, err
		}
		page, err := lc.ossListBucketsRetry(client, oss.Marker(c.Marker), oss.MaxKeys(c.PageSize), lc.payerOption)
		if err != nil {
			return nil, "", "", false, err
		}
		for _, b := range page.Buckets {
			items = append(items, machineListItem{Kind: "bucket", Bucket: b.Name, Region: b.Location, StorageClass: b.StorageClass, Created: &b.CreationDate})
		}
		return items, page.NextMarker, "", page.IsTruncated, nil
	}
	bucket, err := lc.command.ossBucket(target.bucket)
	if err != nil {
		return nil, "", "", false, err
	}
	delimiter := ""
	if directory {
		delimiter = "/"
	}
	options := []oss.Option{oss.Prefix(target.object), oss.Delimiter(delimiter), lc.payerOption}
	var prefixes []string
	var next, secondary string
	var truncated bool
	switch stage {
	case "objects":
		page, e := lc.command.ossListObjectsRetry(bucket, append(options, oss.Marker(c.Marker), oss.MaxKeys(c.PageSize))...)
		if e != nil {
			return nil, "", "", false, e
		}
		for _, o := range page.Objects {
			items = append(items, machineListItem{Kind: "object", Bucket: target.bucket, Key: o.Key, Size: &o.Size, ETag: strings.Trim(o.ETag, "\""), StorageClass: o.StorageClass, Modified: &o.LastModified})
		}
		prefixes = page.CommonPrefixes
		next = page.NextMarker
		truncated = page.IsTruncated
	case "versions":
		page, e := lc.command.ossListObjectVersionsRetry(bucket, append(options, oss.KeyMarker(c.Marker), oss.VersionIdMarker(c.Secondary), oss.MaxKeys(c.PageSize))...)
		if e != nil {
			return nil, "", "", false, e
		}
		for _, o := range page.ObjectDeleteMarkers {
			items = append(items, machineListItem{Kind: "delete_marker", Bucket: target.bucket, Key: o.Key, VersionID: o.VersionId, Latest: &o.IsLatest, Modified: &o.LastModified})
		}
		for _, o := range page.ObjectVersions {
			items = append(items, machineListItem{Kind: "version", Bucket: target.bucket, Key: o.Key, VersionID: o.VersionId, Latest: &o.IsLatest, Size: &o.Size, ETag: strings.Trim(o.ETag, "\""), StorageClass: o.StorageClass, Modified: &o.LastModified})
		}
		prefixes = page.CommonPrefixes
		next = page.NextKeyMarker
		secondary = page.NextVersionIdMarker
		truncated = page.IsTruncated
	case "uploads":
		page, e := lc.command.ossListMultipartUploadsRetry(bucket, append(options, oss.KeyMarker(c.Marker), oss.UploadIDMarker(c.Secondary), oss.MaxUploads(c.PageSize))...)
		if e != nil {
			return nil, "", "", false, e
		}
		for _, o := range page.Uploads {
			items = append(items, machineListItem{Kind: "upload", Bucket: target.bucket, Key: o.Key, UploadID: o.UploadID, Created: &o.Initiated})
		}
		prefixes = page.CommonPrefixes
		next = page.NextKeyMarker
		secondary = page.NextUploadIDMarker
		truncated = page.IsTruncated
	}
	for _, prefix := range prefixes {
		items = append(items, machineListItem{Kind: "prefix", Bucket: target.bucket, Key: prefix})
	}
	return items, next, secondary, truncated, nil
}
