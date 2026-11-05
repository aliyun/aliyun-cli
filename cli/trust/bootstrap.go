package trust

// Public trust anchors shipped with every build. These values are public;
// Root and Recovery private keys are not here. Rotate them by editing this file.
var (
	BuiltinRootEnvelope = `{"payload":"eyJhdXRob3JpdHkiOiJodHRwczovL2FwaS5hbGl5dW4uY29tIiwiZXhwaXJlc19hdCI6Ijk5OTktMTItMzFUMDA6MDA6MDBaIiwia2V5cyI6W3siYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjExMDQwNzI0MTAtYSIsInB1YmxpY19rZXkiOiJEOWkrZXEwVW5XQUNvTUF2MHVNeVYwTmUraWxHcnBrZ1dWclhMK3JGa0JnPSJ9LHsiYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjExMDQwNzI0MTAtYiIsInB1YmxpY19rZXkiOiIzTHg2L2JRRFFGMXFPaUxUcFFsaGY0dks1UE41d1lyeHVJTURMc2Qyei9NPSJ9LHsiYWxnb3JpdGhtIjoiRWQyNTUxOSIsImtleV9pZCI6InJvb3QtMjAyNjExMDQwNzI0MTAtYyIsInB1YmxpY19rZXkiOiJob0VHOHl5N3pOL2cvUDNYMnNBWFpaK1RtOU1naGVtcVA1eTRubXlVVFVjPSJ9XSwicm9sZXMiOnsiYXJ0aWZhY3Qta2V5cyI6eyJrZXlfaWRzIjpbInJvb3QtMjAyNjExMDQwNzI0MTAtYSIsInJvb3QtMjAyNjExMDQwNzI0MTAtYiIsInJvb3QtMjAyNjExMDQwNzI0MTAtYyJdLCJ0aHJlc2hvbGQiOjJ9LCJyb290Ijp7ImtleV9pZHMiOlsicm9vdC0yMDI2MTEwNDA3MjQxMC1hIiwicm9vdC0yMDI2MTEwNDA3MjQxMC1iIiwicm9vdC0yMDI2MTEwNDA3MjQxMC1jIl0sInRocmVzaG9sZCI6Mn19LCJzY2hlbWFfdmVyc2lvbiI6MSwidHlwZSI6InJvb3QiLCJ2ZXJzaW9uIjoxfQ==","schema_version":1,"signatures":[{"algorithm":"Ed25519","key_id":"root-20261104072410-a","signature":"RttMFU283x299rbC8YWzHO0G2RhDaRZobm3hJ6ckgsbNMBHinhaBhcVblpqZ/Q6MihHk9yfIdXT2sfRQTrDLAw=="},{"algorithm":"Ed25519","key_id":"root-20261104072410-b","signature":"TF26Ry0rPtv8m9JhWlF8gTdmD1R3ZPDkFC/E7h5D4ZrNPXkpGP4aveHEkV35A0YJGyQzQhbE0LMlLJ4BeCNXBg=="},{"algorithm":"Ed25519","key_id":"root-20261104072410-c","signature":"Pcu/hmNiPpbjXmC1fVX9XC5nJlRkqycQYxSdVrxTCfuYV3YxUXIYBvC8gAfPvR7jq8qoqCG7RdPPuYgI45uTAw=="}]}
`
	BuiltinRecoveryKeysJSON = `[{"algorithm":"Ed25519","key_id":"recovery-20261104072410-a","public_key":"gPh9hBq5HElW3tgIey/hnW19Z9vFmyWyf0LL2tYMLr8="},{"algorithm":"Ed25519","key_id":"recovery-20261104072410-b","public_key":"Nvk6iGzrlolr9xsC7jalE+WYA9cX9RXU776yYYT3z+w="},{"algorithm":"Ed25519","key_id":"recovery-20261104072410-c","public_key":"qrKTJTxR7cQ00P9PsBvHG/v90AiRSgPJiZ/oVVVo23g="}]
`
	BuiltinRecoveryRoleJSON = `{"key_ids":["recovery-20261104072410-a","recovery-20261104072410-b","recovery-20261104072410-c"],"threshold":2}
`
)

const (
	DiscoveryURLCN   = "https://api.aliyun.com/.well-known/aliyun-cli-trust.json"
	DiscoveryURLIntl = "https://api.alibabacloud.com/.well-known/aliyun-cli-trust.json"
	// DiscoveryURL remains as the compatibility name for the default cn site.
	DiscoveryURL = DiscoveryURLCN
)

var MetadataHostAllowlist = []string{
	"api.aliyun.com", "api.alibabacloud.com", "aliyuncli.alicdn.com",
}
var RecoveryHostAllowlist = []string{
	"api.aliyun.com", "api.alibabacloud.com", "aliyuncli.alicdn.com",
}
