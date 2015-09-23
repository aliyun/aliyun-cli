#!/usr/bin/env python
#coding=utf-8

# Copyright (C) 2011, Alibaba Cloud Computing

#Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

#The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

#THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

from oss.oss_api import *
from oss.oss_util import *
from oss.oss_xml_handler import *
from  aliyunCliParser import aliyunCliParser

import signal
import ConfigParser
from optparse import OptionParser
from optparse import Values
import os
import re
import time
import Queue
import sys
import socket
import shutil
reload(sys)
sys.setdefaultencoding("utf-8")

CMD_LIST = {}
HELP_CMD_LIST = ['--help','-h','help']
ACL_LIST = ['private', 'public-read', 'public-read-write']
OSS_PREFIX = 'oss://'
CONFIGFILE = "%s/.aliyuncli/osscredentials" % os.path.expanduser('~')
CONFIGSECTION = 'OSSCredentials'
DEFAUL_HOST = "oss.aliyuncs.com"
OSS_HOST = DEFAUL_HOST
ID = ""
KEY = ""
STS_TOKEN = None

TOTAL_PUT = AtomicInt()
PUT_OK = AtomicInt()
PUT_FAIL = AtomicInt()
PUT_SKIP = AtomicInt()
FILE_NUM_TOTAL = AtomicInt()
FILE_NUM_OK = AtomicInt()

GET_OK = AtomicInt()
GET_FAIL = AtomicInt()
GET_SKIP = AtomicInt()

DELETE_OK = AtomicInt()
COPY_OK = AtomicInt()

SEND_BUF_SIZE = 8192
RECV_BUF_SIZE = 1024*1024*10
MAX_OBJECT_SIZE = 5*1024*1024*1024
MAX_RETRY_TIMES = 3
IS_DEBUG = False
ERROR_FILE_LIST = []
AUTO_DUMP_FILE_NUM = 50

RET_OK = 0
RET_FAIL = -1
RET_SKIP = 1

lock = threading.Lock()

HELP = \
'''The valid command as follows::
    GetAllBucket
    CreateBucket              oss://bucket --acl [acl] --location [location]
    DeleteBucket              oss://bucket
    DeleteWholeBucket         oss://bucket
    GetBucketLocation         oss://bucket
    PutBucketCors             oss://bucket localfile
    GetBucketCors             oss://bucket
    DeleteBucketCors          oss://bucket
    PutBucketLogging          oss://source_bucket oss://target_bucket/[prefix]
    GetBucketLogging          oss://bucket
    DeleteBucketLogging       oss://bucket
    PutBucketWebsite          oss://bucket indexfile [errorfile]
    GetBucketWebsite          oss://bucket
    DeleteBucketWebsite       oss://bucket
    PutBucketLifeCycle        oss://bucket localfile
    GetBucketLifeCycle        oss://bucket
    DeleteBucketLifeCycle     oss://bucket
    PutBucketReferer          oss://bucket --allow_empty_referer true --referer "referer1,referer2,...,refererN"
    GetBucketReferer          oss://bucket
    GetAcl                    oss://bucket
    SetAcl                    oss://bucket --acl [acl]
                              allow private, public-read, public-read-write

    List                      oss://bucket/[prefix] [marker] [delimiter] [maxkeys]
                              oss://bucket/[prefix] --marker xxx --delimiter xxx --maxkeys xxx
    MkDir                     oss://bucket/dirname
    ListAllObject             oss://bucket/[prefix]
    ListAllDir                oss://bucket/[prefix]
    DeleteAllObject           oss://bucket/[prefix] --force false
    DownloadAllObject         oss://bucket/[prefix] localdir --replace false --thread_num 5
    DownloadToDir             oss://bucket/[prefix] localdir --replace false --temp_dir xxx --thread_num 5
    UploadObjectFromLocalDir  localdir oss://bucket/[prefix] --check_point check_point_file --replace false --check_md5 false --thread_num 5
    Put                       oss://bucket/object --content_type [content_type] --headers \"key1:value1#key2:value2\" --check_md5 false

    Get                 oss://bucket/object localfile
    MultiGet            oss://bucket/object localfile --thread_num 5
    Cat                 oss://bucket/object
    Meta                oss://bucket/object
    Info                oss://bucket/object
    Copy                oss://source_bucket/source_object oss://target_bucket/target_object --headers \"key1:value1#key2:value2\"
    CopyLargeFile       oss://source_bucket/source_object oss://target_bucket/target_object --part_size 10*1024*1024 --upload_id xxx
    CopyBucket          oss://source_bucket/[prefix] oss://target_bucket/[prefix] --headers \"key1:value1\" --replace false
    Delete              oss://bucket/object
    SignUrl             oss://bucket/object --timeout [timeout_seconds]
    CreateLinkFromFile  oss://bucket/object object_name_list_file
    CreateLink          oss://bucket/object object1 object2 ... objectN
    GetLinkIndex        oss://bucket/object
    Options             oss://bucket/[object] --origin xxx --method [GET, PUT, DELETE, HEAD, POST]
    UploadDisk          localdir oss://bucket/[prefix] [--check_point check_point_file --filename filename_file --replace false --content_type xxx --skip_dir false --skip_suffix false --out xxx] --device_id xxx --check_md5 false

    Init                oss://bucket/object
    ListPart            oss://bucket/object --upload_id xxx
    ListParts           oss://bucket
    GetAllPartSize      oss://bucket
    Cancel              oss://bucket/object --upload_id xxx

    MultiUpload          localfile oss://bucket/object --upload_id xxx --thread_num 10 --max_part_num 1000 --check_md5 false
    UploadPartFromFile   localfile oss://bucket/object --upload_id xxx --part_number xxx
    UploadPartFromString   oss://bucket/object --upload_id xxx --part_number xxx --data xxx
    Config                 --host oss.aliyuncs.com --accessid accessid --accesskey accesskey --sts_token token
    '''

def print_result(cmd, res):
    '''
    Print HTTP Response if failedd.
    '''
    try:
        if res.status / 100 == 2:
            pass
        else:
            body = res.read()
            print "Error Headers:\n"
            print res.getheaders()
            print "Error Body:\n"
            print body[0:1024]
            print "Error Status:\n"
            print res.status
            print cmd, "Failed!"
            if res.status == 403:
                check_endpoint_error(body)
            exit(-1)
    except AttributeError:
        pass

def format_size(size):
    size = float(size)
    coeffs = ['K', 'M', 'G', 'T']
    coeff = ""
    while size > 2048:
        size /= 1024
        coeff = coeffs.pop(0)
    return str("%.2f"%size) + coeff + "B"

def format_utf8(string):
    string = smart_code(string)
    if isinstance(string, unicode):
        string = string.encode('utf-8')                
    return string

def split_path(path):
    if not path.lower().startswith(OSS_PREFIX):
        print "%s parameter %s invalid, " \
              "must be start with %s" % \
              (args[0], args[1], OSS_PREFIX)
        sys.exit(1)
    pather = path[len(OSS_PREFIX):].split('/')
    return pather

def check_upload_id(upload_id):
    upload_id_len = 32 
    if len(upload_id) != upload_id_len:
        print "upload_id is a 32-bit string generated by OSS"
        print "you can get valid upload_id by init or listparts command"
        sys.exit(1)

def check_bucket(bucket):
    if len(bucket) == 0:
        print "Bucket should not be empty!"
        print "Please input oss://bucket"
        sys.exit(1)

def check_object(object):
    if len(object) == 0:
        print "Object should not be empty!"
        print "Please input oss://bucket/object"
        sys.exit(1)

    if object.startswith("/"):
        print "object name should not begin with / "
        sys.exit(-1)

def check_localfile(localfile):
    if not os.path.isfile(localfile):
        print "%s is not existed!" % localfile
        sys.exit(1)

def check_args(argv, args=None):
    if not args:
        args = []
    if len(args) < argv:
        print "%s miss parameters" % args[0]
        sys.exit(1)

def check_bucket_object(bucket, object):
    check_bucket(bucket)
    check_object(object)

def parse_bucket_object(path):
    pather = split_path(path)
    bucket = ""
    object = ""
    if len(pather) > 0:
        bucket = pather[0]
    if len(pather) > 1:
        object += '/'.join(pather[1:])
    object = smart_code(object)
    if object.startswith("/"):
        print "object name SHOULD NOT begin with /"
        sys.exit(1)
    return (bucket, object)

def parse_bucket(path):
    bucket = path
    if bucket.startswith(OSS_PREFIX):
        bucket = bucket[len(OSS_PREFIX):]
    tmp_list = bucket.split("/")
    if len(tmp_list) > 0:
        bucket = tmp_list[0]
    return bucket

def check_endpoint_error(xml_string):
    try:
        xml = minidom.parseString(xml_string)
        end_point = get_tag_text(xml, 'Endpoint')
        if end_point:
            print 'You should send all request to %s' % end_point
    except:
        pass

def cmd_listing(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    (bucket, object) = parse_bucket_object(args[1])
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    prefix = object
    marker = ''
    delimiter = ''
    maxkeys = 1000
    if options.marker:
        marker = options.marker
    if options.delimiter:
        delimiter = options.delimiter
    if options.maxkeys:
        maxkeys = options.maxkeys
    if len(args) == 3: 
        marker = args[2]
    elif len(args) == 4:
        marker = args[2]
        delimiter = args[3]
    elif len(args) >= 5:
        marker = args[2]
        delimiter = args[3]
        maxkeys = args[4]

    prefix = smart_code(prefix)
    marker = smart_code(marker)
    delimiter = smart_code(delimiter)
    maxkeys = smart_code(maxkeys)
    exclude = options.exclude
    res = get_oss().get_bucket(bucket, prefix, marker, delimiter, maxkeys)
    if (res.status / 100) == 2:
        body = res.read()
        hh = GetBucketXml(body)
        (fl, pl) = hh.list()
        print "prefix list is: "
        for i in pl:
            if exclude and i.startswith(exclude):
                continue
            print i
        print "object list is: "
        for i in fl:
            if len(i) == 7:
                try:
                    if exclude and i[0].startswith(exclude):
                        continue
                    print "%16s %6s %8s %s/%s" % (convert_to_localtime(i[1]), format_size((int)(i[3])), i[6], OSS_PREFIX + bucket, i[0])
                except:
                    print "Exception when print :", i
        print "\nprefix list number is: %s " % len(pl)
        print "object list number is: %s " % len(fl)
    return res

def cmd_listparts(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    (bucket, object) = parse_bucket_object(args[1])
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    print "%20s %20s %20s" % ("UploadId", "Path", "InitTime")
    for i in get_all_upload_id_list(get_oss(), bucket, object):
        print "%20s oss://%s/%s %20s" % (i[1], bucket, i[0], convert_to_localtime(i[2]))

def cmd_getallpartsize(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    (bucket, object) = parse_bucket_object(args[1])
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    total_part_size = 0
    print "%5s %20s %20s %s" % ("Number", "UploadId", "Size", "Path")
    for i in get_all_upload_id_list(get_oss(), bucket):
        upload_id = i[1]
        object = i[0]
        for i in get_part_list(get_oss(), bucket, object, upload_id):
            part_size = (int)(i[2])
            total_part_size += part_size 
            print "%5s %20s %10s oss://%s/%s" % (i[0], upload_id, format_size(part_size), bucket, object) 
    print "totalsize is: real:%s, format:%s " % (total_part_size, format_size(total_part_size))

def cmd_init_upload(args, options):
    check_args(2, args)  
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    check_bucket_object(bucket, object)
    upload_id = get_upload_id(get_oss(), bucket, object)
    print 'Upload Id: %s' % (upload_id)

def cmd_listpart(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    if options.upload_id is None:
        print "upload_id invalid, please set with --upload_id=xxx"
        sys.exit(1)
    print "%5s %32s %20s %20s" % ("PartNumber".ljust(10), "ETag".ljust(34), "Size".ljust(20), "LastModifyTime".ljust(32))
    for i in get_part_list(get_oss(), bucket, object, options.upload_id):
        if len(i) >= 4:
            print "%s %s %s %s" % (str(i[0]).ljust(10), str(i[1]).ljust(34), str(i[2]).ljust(20), str(i[3]).ljust(32))

def cmd_upload_part_from_file(args, options):
    check_args(3, args)
    localfile = args[1]
    check_localfile(localfile)
    path = args[2]
    (bucket, object) = parse_bucket_object(path)
    check_bucket_object(bucket, object)
    if options.upload_id is None:
        print "upload_id invalid, please set with --upload_id=xxx"
        sys.exit(1)
    if options.part_number is None:
        print "part_number invalid, please set with --part_number=xxx"
        sys.exit(1)
    res = get_oss().upload_part(bucket, object, localfile, options.upload_id, options.part_number)
    return res

def cmd_upload_part_from_string(args, options):
    check_args(2, args)
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    check_bucket_object(bucket, object)
    if options.upload_id is None:
        print "upload_id invalid, please set with --upload_id=xxx"
        sys.exit(1)
    if options.part_number is None:
        print "part_number invalid, please set with --part_number=xxx"
        sys.exit(1)
    if options.data is None:
        print "data invalid, please set with --data=xxx"
        sys.exit(1)
    res = get_oss().upload_part_from_string(bucket, object, options.data, options.upload_id, options.part_number)
    return res

def cmd_listallobject(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    prefix = object 
    marker = ""
    total_object_num = 0
    totalsize = 0
    totaltimes = 0
    delimiter = ''
    maxkeys = '1000'
    if options.out:
        f = open(options.out, "w")
    while 1:
        res = get_oss().get_bucket(bucket, prefix, marker, delimiter, maxkeys)
        if res.status != 200:
            return res
        body = res.read()
        (tmp_object_list, marker) = get_object_list_marker_from_xml(body)
        for i in tmp_object_list:
            object = i[0]
            length = i[1]
            last_modify_time = i[2]
            total_object_num += 1
            totalsize += (int)(length)
            if options.exclude:
                exclude = options.exclude
                if object.startswith(exclude):
                    continue
            msg = "%s%s/%s" % (OSS_PREFIX, bucket, object)
            print "%16s %6s %s/%s " % (convert_to_localtime(last_modify_time), format_size(length), OSS_PREFIX + bucket, object)
            if options.out:
                f.write(msg)
                f.write("\n")
        totaltimes += 1
        if len(marker) == 0:
            break
    if options.out:
        f.close()
        print "the object list result is saved into %s" % options.out
    print "object list number is: %s " % total_object_num
    print "totalsize is: real:%s, format:%s " % (totalsize, format_size(totalsize))
    print "request times is: %s" % totaltimes
    return res

def cmd_listalldir(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    prefix = object 
    if prefix and not prefix.endswith("/"):
        prefix = "%s/" % prefix
    marker = ""
    total_object_num = 0
    totalsize = 0
    totaltimes = 0
    delimiter = '/'
    maxkeys = '1000'
    while 1:
        res = get_oss().get_bucket(bucket, prefix, marker, delimiter, maxkeys)
        if res.status != 200:
            return res
        body = res.read()
        (tmp_object_list, marker) = get_dir_list_marker_from_xml(body)
        for i in tmp_object_list:
            if i.endswith("/"):
                i = i[:-1]
            msg = "%s" % (os.path.basename(i))
            print msg
            total_object_num += 1
        totaltimes += 1
        if len(marker) == 0:
            break
    print "\ncommon prefix list number is: %s " % total_object_num
    print "request times is: %s" % totaltimes
    return res

def get_object(bucket, object, object_prefix, local_path, length, last_modify_time, replace, retry_times = MAX_RETRY_TIMES, temp_dir = None):
    '''
    return RET_OK, RET_FAIL, RET_SKIP
    '''
    show_bar = False
    object = smart_code(object)
    tmp_object = object
    if object_prefix == object[:len(object_prefix)]:
        tmp_object = object[len(object_prefix):]
    while 1:
        if not tmp_object.startswith("/"):
            break
        tmp_object = tmp_object[1:]
    localfile = os.path.join(local_path, tmp_object) 
    localfile = smart_code(localfile)

    temp_filename = ''
    if temp_dir:
        temp_filename = get_unique_temp_filename(temp_dir, localfile)

    for i in xrange(retry_times):
        try:
            if os.path.isfile(localfile):
                if replace:
                    os.remove(localfile)
                else:
                    t1 = last_modify_time
                    t2 = (int)(os.path.getmtime(localfile))
                    if (int)(length) == os.path.getsize(localfile) and t1 < t2:
                        #skip download this object these conditions match
                        print "no need to get %s/%s to %s" % (bucket, object, localfile)
                        return RET_SKIP
            else:
                try:
                    dirname = os.path.dirname(localfile)
                    if not os.path.isdir(dirname):
                        os.makedirs(dirname)
                    if temp_dir:
                        dirname = os.path.dirname(temp_filename)
                        if not os.path.isdir(dirname):
                            os.makedirs(dirname)
                except:
                    pass
            filename = localfile
            if temp_dir:
                filename = temp_filename
            if os.path.isdir(filename):
                print "no need to get %s/%s to %s" % (bucket, object, filename)
                return RET_SKIP
            ret = continue_get(bucket, object, filename)
            if ret:
                print "get %s/%s to %s OK" % (bucket, object, localfile)
                if temp_dir:
                    shutil.move(temp_filename, localfile)
                    pass
                return RET_OK
            else:
                print "get %s/%s to %s FAIL" % (bucket, object, localfile)
        except:
            print "get %s/%s to %s exception" % (bucket, object, localfile)
            print sys.exc_info()[0], sys.exc_info()[1]
    os.remove(temp_filename)
    return RET_FAIL

class DownloadObjectWorker(threading.Thread):
    def __init__(self, retry_times, queue):
        threading.Thread.__init__(self)
        self.queue = queue
        self.retry_times = retry_times
        self.ok_num = 0
        self.fail_num = 0
        self.skip_num = 0

    def run(self):
        while 1:
            try:
                (get_object, bucket, object, object_prefix, local_path, length, last_modify_time, replace, retry_times, temp_dir) = self.queue.get(block=False)

                ret = get_object(bucket, object, object_prefix, local_path, length, last_modify_time, replace, self.retry_times, temp_dir)
                if ret == RET_OK: 
                    self.ok_num += 1
                elif ret == RET_SKIP:
                    self.skip_num += 1
                else:
                    self.fail_num += 1
                self.queue.task_done()
            except Queue.Empty:
                break
            except:
                self.fail_num += 1
                print sys.exc_info()[0], sys.exc_info()[1]
                self.queue.task_done()
        global GET_SKIP
        global GET_OK
        global GET_FAIL
        lock.acquire()
        GET_SKIP += self.skip_num
        GET_OK += self.ok_num
        GET_FAIL += self.fail_num
        lock.release()

def cmd_downloadallobject(args, options):
    check_args(3, args)  
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    check_bucket(bucket)
    local_path = args[2]
    if os.path.isfile(local_path):
        print "%s is not dir, please input localdir" % local_path
        exit(-1)
    replace = False
    if options.replace is not None and options.replace.lower() == "true":
        replace = True
    prefix = object
    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES
    if options.retry_times:
        retry_times = (int)(options.retry_times)

    temp_dir = None
    if options.temp_dir:
        temp_dir = options.temp_dir
        if not os.path.exists(temp_dir):
            os.makedirs(temp_dir)

    marker = ""
    delimiter = ''
    maxkeys = '1000'
    handled_obj_num = 0
    while 1:
        queue = Queue.Queue(0)
        for i in xrange(0, retry_times):
            res = get_oss().get_bucket(bucket, prefix, marker, delimiter, maxkeys)
            if res.status/100 == 5:
                continue
            else:
                break
        if res.status != 200:
            return res
        body = res.read()
        (tmp_object_list, marker) = get_object_list_marker_from_xml(body)
        for i in tmp_object_list:
            object = i[0]
            length = i[1]
            last_modify_time = format_unixtime(i[2])
            if str(length) == "0" and object.endswith("/"):
                continue
            handled_obj_num += 1 
            queue.put((get_object, bucket, object, prefix, local_path, length, last_modify_time, replace, MAX_RETRY_TIMES, temp_dir))
        thread_pool = []
        for i in xrange(thread_num):
            current = DownloadObjectWorker(retry_times, queue)
            thread_pool.append(current)
            current.start()
        queue.join()
        for item in thread_pool:
            item.join()
        if len(marker) == 0:
            break
    global GET_OK
    global GET_SKIP
    global GET_FAIL
    print "Total being downloaded objects num: %s, they are downloaded into %s" % (GET_OK + GET_FAIL + GET_SKIP, local_path)
    print "OK num:%s, SKIP num:%s, FAIL num:%s" % (GET_OK, GET_SKIP, GET_FAIL)
    if temp_dir and os.path.abspath(local_path) != os.path.abspath(temp_dir):
        shutil.rmtree(temp_dir, True)
    if GET_FAIL != 0:
        exit(-1)

def put_object(bucket, object, local_file, local_modify_time, is_replace, is_check_md5=False, content_type="", multipart_threshold=100*1024*1024, retry_times=2):
    '''
    return RET_OK, RET_FAIL, RET_SKIP
    '''
    if not os.path.isfile(local_file):
        print "upload %s FAIL, no such file." % (local_file)
        return RET_FAIL
    show_bar = False
    oss = get_oss(show_bar)
    object = smart_code(object)
    if len(object) == 0:
        print "object is empty when put /%s/%s, skip" % (bucket, object)
        return RET_SKIP
    local_file_size = os.path.getsize(local_file)
    if not is_replace:
        try:
            res = oss.head_object(bucket, object)
            if res.status == 200 and str(local_file_size) == res.getheader('content-length'):
                oss_gmt = res.getheader('last-modified')
                format = "%a, %d %b %Y %H:%M:%S GMT"
                oss_last_modify_time = format_unixtime(oss_gmt, format)
                if not local_modify_time:
                    local_modify_time = (int)(os.path.getmtime(local_file))
                if oss_last_modify_time >= local_modify_time:
                    #print "upload %s is skipped" % (local_file)
                    return RET_SKIP
        except:
            print "%s %s" % (sys.exc_info()[0], sys.exc_info()[1])
    if is_check_md5:
        md5string, base64md5 = get_file_md5(local_file)
    for i in xrange(retry_times):
        try:
            if local_file_size > multipart_threshold:
                upload_id = ""
                thread_num = 5
                max_part_num = 10000
                headers = {}
                if is_check_md5:
                    headers['x-oss-meta-md5'] = md5string
                if content_type:
                    headers['Content-Type'] = content_type
                res = oss.multi_upload_file(bucket, object, local_file, upload_id, thread_num, max_part_num, headers, check_md5=is_check_md5)
            else:
                headers = {}
                if is_check_md5:
                    headers['Content-MD5'] = base64md5
                    headers['x-oss-meta-md5'] = md5string
                res = oss.put_object_from_file(bucket, object, local_file, content_type, headers)
            if 200 == res.status:
                return RET_OK
            else:
                print "upload %s to /%s/%s FAIL, status:%s, request-id:%s" % (local_file, bucket, object, res.status, res.getheader("x-oss-request-id"))
        except:
            print "upload %s/%s from %s exception" % (bucket, object, local_file)
            print sys.exc_info()[0], sys.exc_info()[1]
    return RET_FAIL

class UploadObjectWorker(threading.Thread):
    def __init__(self, check_point_file, retry_times, queue):
        threading.Thread.__init__(self)
        self.check_point_file = check_point_file
        self.queue = queue
        self.file_time_map = {}
        self.error_file_list = []
        self.retry_times = retry_times
        self.ok_num = 0
        self.fail_num = 0
        self.skip_num = 0

    def run(self):
        global PUT_SKIP
        global PUT_OK
        global PUT_FAIL
        global TOTAL_PUT
        global FILE_NUM_OK
        while 1:
            try:
                (put_object, bucket, object, local_file, local_modify_time, is_replace, is_check_md5, content_type, multipart_threshold) = self.queue.get(block=False)
                ret = put_object(bucket, object, local_file, local_modify_time, is_replace, is_check_md5, content_type, multipart_threshold, self.retry_times)
                is_ok = False
                if ret == RET_OK: 
                    is_ok = True
                    self.ok_num += 1
                    PUT_OK += 1
                    FILE_NUM_OK += 1
                elif ret == RET_SKIP:
                    is_ok = True
                    self.skip_num += 1
                    PUT_SKIP += 1
                    FILE_NUM_OK += 1
                else:
                    self.fail_num += 1
                    PUT_FAIL += 1
                    self.error_file_list.append(local_file)
                if is_ok:
                    local_file_full_path = os.path.abspath(local_file)
                    local_file_full_path = format_utf8(local_file_full_path)
                    self.file_time_map[local_file_full_path] = (int)(os.path.getmtime(local_file))
                
                sum = (PUT_SKIP + PUT_OK + PUT_FAIL)
                if TOTAL_PUT > 0:
                    exec("rate = 100*%s/(%s*1.0)" % (sum, TOTAL_PUT))
                else:
                    rate = 0
                print '\rOK:%s, FAIL:%s, SKIP:%s, TOTAL_DONE:%s, TOTAL_TO_DO:%s, PROCESS:%.2f%%' % (PUT_OK, PUT_FAIL, PUT_SKIP, sum, TOTAL_PUT, rate),
                sys.stdout.flush()

                if self.ok_num % AUTO_DUMP_FILE_NUM == 0:
                    if len(self.file_time_map) != 0:
                        dump_check_point(self.check_point_file, self.file_time_map)
                        self.file_time_map = {}

                self.queue.task_done()
            except Queue.Empty:
                break
            except:
                PUT_FAIL += 1
                print sys.exc_info()[0], sys.exc_info()[1]
                self.queue.task_done()
        
        if len(self.error_file_list) != 0:
            lock.acquire()
            ERROR_FILE_LIST.extend(self.error_file_list)
            lock.release()
        if len(self.file_time_map) != 0:
            dump_check_point(self.check_point_file, self.file_time_map)
            
def load_check_point(check_point_file):
    file_time_map = {}
    if os.path.isfile(check_point_file):
        f = open(check_point_file)
        for line in f:
            line = line.strip()
            tmp_list = line.split('#')
            if len(tmp_list) > 1:
                time_stamp = (float)(tmp_list[0])
                time_stamp = (int)(time_stamp)
                #file_name = "".join(tmp_list[1:])
                file_name = line[len(tmp_list[0])+1:]
                file_name = format_utf8(file_name)
                if file_time_map.has_key(file_name) and file_time_map[file_name] > time_stamp:
                    continue
                file_time_map[file_name] = time_stamp
        f.close()
    return file_time_map

def load_filename(filename_file):
    filenames = []
    if os.path.isfile(filename_file):
        f = open(filename_file)
        for line in f:
            line = line.strip()
            filenames.append(line)
    return filenames

def dump_filename(filename_file, filenames=None):
    if len(filename_file) == 0 or len(filenames) == 0:
        return
    try:
        f = open(filename_file,"w")
        for filename in filenames:
            line = "%s\n" %(filename)
            f.write(line)
    except:
        pass
    try:
        f.close()
    except:
        pass        

def dump_check_point(check_point_file, result_map=None):
    if len(check_point_file) == 0 or len(result_map) == 0:
        return 
    lock.acquire()
    old_file_time_map = {}
    if os.path.isfile(check_point_file):
        old_file_time_map = load_check_point(check_point_file)
    try:
        f = open(check_point_file,"w")
        for k, v in result_map.items():
            if old_file_time_map.has_key(k) and old_file_time_map[k] < v:
                del old_file_time_map[k]
            line = "%s#%s\n" % (v, k)
            line = format_utf8(line)
            f.write(line)
        for k, v in old_file_time_map.items():
            line = "%s#%s\n" % (v, k)
            line = format_utf8(line)
            f.write(line)
    except:
        pass
    try:
        f.close()
    except:
        pass
    lock.release()

def format_object(object):
    tmp_list = object.split(os.sep)
    object = "/".join(x for x in tmp_list if x.strip() and x != "/")
    while 1:
        if object.find('//') == -1:
            break
        object = object.replace('//', '/')
    return  object

def get_object_name(filename, filepath):
    filename = format_object(filename)
    filepath = format_object(filepath)
    file_name = os.path.basename(filename)
    return file_name

def get_file_names_from_disk(path, topdown):
    filenames = []
    for root, dirs, files in os.walk(path, topdown):
        for filespath in files:
            filename = os.path.join(root, filespath)
            filenames.append(filename)
    return filenames

#for offline uploadfile to oss
def cmd_upload_disk(args, options):
    check_args(3, args)
    path = args[2]
    (bucket, object) = parse_bucket_object(path)
    check_bucket(bucket)
    local_path = args[1]

    if not os.path.isdir(local_path):
        print "%s is not dir, please input localdir" % local_path
        exit(-1)
    if not local_path.endswith(os.sep):     
        local_path = "%s%s" % (local_path, os.sep)
    if not options.device_id:
        print "please set device id with --device_id=xxx"
        exit(-1)
    
    check_point_file = ""
    is_check_point = False
    file_time_map = {}
    if options.check_point:
        is_check_point = True
        check_point_file = options.check_point
        file_time_map = load_check_point(check_point_file)    

    filename_file = ""
    filenames = []
    is_filename_file = False
    if options.filename_list:
        filename_file = options.filename_list
        if os.path.isfile(filename_file):
            is_filename_file = True
            filenames = load_filename(filename_file)
 
    prefix = object
    is_replace = False
    if options.replace:
        if options.replace.lower() == "true":
            is_replace = True
    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES 
    if options.retry_times:
        retry_times = (int)(options.retry_times)

    is_check_md5 = False
    if options.check_md5:
        if options.check_md5.lower() == "true":
            is_check_md5 = True
    multipart_threshold = 100*1024*1024
    if options.multipart_threshold:
        multipart_threshold = (int)(options.multipart_threshold)

    total_upload_num = 0
    topdown = True
    def process_localfile(items):
        queue = Queue.Queue(0)
        for local_file in items:
            if os.path.isfile(local_file):
                local_modify_time = 0
                local_file_full_path = os.path.abspath(local_file) 
                local_file_full_path = format_utf8(local_file_full_path)
                if is_check_point and file_time_map.has_key(local_file_full_path):
                    local_modify_time = (int)(os.path.getmtime(local_file))
                    record_modify_time = file_time_map[local_file_full_path]
                    if local_modify_time <= record_modify_time:
                        print 'file:%s already upload' %(local_file_full_path)
                        global FILE_NUM_OK
                        FILE_NUM_OK += 1
                        continue
                if options.skip_dir and options.skip_dir.lower() == "true":
                    object = smart_code(os.path.basename(local_file))
                else:
                    object = smart_code(local_file)
                if options.strip_dir:
                    strip_dir = options.strip_dir
                    if not strip_dir.endswith("/"):
                        strip_dir = "%s/" % strip_dir
                    if object.startswith(strip_dir):
                        object = object[len(options.strip_dir):]
                if options.skip_suffix and options.skip_suffix.lower() == "true":
                    pos = object.rfind(".") 
                    if pos != -1:
                        object = object[:pos]
                while 1:
                    if object.startswith("/"):
                        object = object[1:]
                    else:
                        break
                if prefix:
                    if prefix.endswith("/"):
                        object = "%s%s" % (prefix, object) 
                    else:
                        object = "%s/%s" % (prefix, object) 
                queue.put((put_object, bucket, object, local_file, local_modify_time, is_replace, is_check_md5, options.content_type, multipart_threshold))
        qsize = queue.qsize()
        global TOTAL_PUT
        TOTAL_PUT += qsize
        thread_pool = []
        for i in xrange(thread_num):
            current = UploadObjectWorker(check_point_file, retry_times, queue)
            thread_pool.append(current)
            current.start()
        queue.join()
        for item in thread_pool:
            item.join()
        return qsize

    if not is_filename_file:
        filenames = get_file_names_from_disk(local_path, topdown);
        dump_filename(filename_file, filenames)
    global FILE_NUM_TOTAL
    FILE_NUM_TOTAL += len(filenames)

    total_upload_num += process_localfile(filenames);
    print ""
    print "DEVICEID:sn%s" % options.device_id
    global PUT_OK
    global PUT_SKIP
    global PUT_FAIL
    print "This time Total being uploaded localfiles num: %s" % (PUT_OK + PUT_SKIP + PUT_FAIL)
    print "This time OK num:%s, SKIP num:%s, FAIL num:%s" % (PUT_OK, PUT_SKIP, PUT_FAIL)
    print "Total file num:%s, OK file num:%s" %(FILE_NUM_TOTAL, FILE_NUM_OK)
    if PUT_FAIL != 0:
        print "FailUploadList:"
        for i in set(ERROR_FILE_LIST):
            print i
        if options.out:
            try:
                f = open(options.out, "w")
                for i in set(ERROR_FILE_LIST):
                    f.write("%s\n" % i.strip())
                f.close()
                print "FailUploadList is written into %s" % options.out
            except:
                print "write upload failed file exception"
                print sys.exc_info()[0], sys.exc_info()[1]
        exit(-1)

def cmd_upload_object_from_localdir(args, options):
    check_args(3, args)
    path = args[2]
    (bucket, object) = parse_bucket_object(path)
    check_bucket(bucket)
    local_path = args[1]
    if not os.path.isdir(local_path):
        print "%s is not dir, please input localdir" % local_path
        exit(-1)
    if not local_path.endswith(os.sep):     
        local_path = "%s%s" % (local_path, os.sep)
        
    is_check_point = False
    check_point_file = ""
    file_time_map = {}
    if options.check_point:
        is_check_point = True
        check_point_file = options.check_point
        file_time_map = load_check_point(check_point_file)

    prefix = object
    is_replace = False
    if options.replace:
        if options.replace.lower() == "true":
            is_replace = True
    is_check_md5 = False
    if options.check_md5:
        if options.check_md5.lower() == "true":
            is_check_md5 = True
    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES 
    if options.retry_times:
        retry_times = (int)(options.retry_times)
    multipart_threshold = 100*1024*1024
    if options.multipart_threshold:
        multipart_threshold = (int)(options.multipart_threshold)

    total_upload_num = 0
    topdown = True
    def process_localfile(items):
        queue = Queue.Queue(0)
        for item in items:
            local_file = os.path.join(root, item) 
            if os.path.isfile(local_file):
                local_file_full_path = os.path.abspath(local_file)
                local_file_full_path = format_utf8(local_file_full_path)
                local_modify_time = 0
                if is_check_point and file_time_map.has_key(local_file_full_path):
                    local_modify_time = (int)(os.path.getmtime(local_file))
                    record_modify_time = file_time_map[local_file_full_path]
                    if local_modify_time <= record_modify_time:
                        continue
                object = get_object_name(smart_code(local_file), smart_code(local_path))
                if prefix:
                    if prefix.endswith("/"):
                        object = "%s%s" % (prefix, object) 
                    else:
                        object = "%s/%s" % (prefix, object) 
                content_type = ''
                queue.put((put_object, bucket, object, local_file, local_modify_time, is_replace, is_check_md5, content_type, multipart_threshold))
        qsize = queue.qsize()
        thread_pool = []
        global TOTAL_PUT
        TOTAL_PUT += qsize
        for i in xrange(thread_num):
            current = UploadObjectWorker(check_point_file, retry_times, queue)
            thread_pool.append(current)
            current.start()
        queue.join()
        for item in thread_pool:
            item.join()
        return qsize
    for root, dirs, files in os.walk(local_path, topdown):
        total_upload_num += process_localfile(files)
        total_upload_num += process_localfile(dirs)
    global PUT_OK
    global PUT_SKIP
    global PUT_FAIL
    print ""
    print "Total being uploaded localfiles num: %s" % (PUT_OK + PUT_SKIP + PUT_FAIL)
    print "OK num:%s, SKIP num:%s, FAIL num:%s" % (PUT_OK, PUT_SKIP, PUT_FAIL)
    if PUT_FAIL != 0:
        exit(-1)

def get_object_list_marker_from_xml(body):
    #return ([(object, object_length, last_modify_time)...], marker)
    object_meta_list = []
    next_marker = ""
    hh = GetBucketXml(body)
    (fl, pl) = hh.list()
    if len(fl) != 0:
        for i in fl:
            object = convert_utf8(i[0])
            last_modify_time = i[1]
            length = i[3]
            object_meta_list.append((object, length, last_modify_time))
    if hh.is_truncated:
        next_marker = hh.nextmarker
    return (object_meta_list, next_marker)

def cmd_deleteallobject(args, options):
    if len(args) == 1:
        return cmd_getallbucket(args, options)
    path = args[1]
    (bucket, object) = parse_bucket_object(path)
    if len(bucket) == 0:
        return cmd_getallbucket(args, options)
    force_delete = False
    if options.force and options.force.lower() == "true":
        force_delete = True

    if not force_delete:
        ans = raw_input("DELETE all objects? Y/N, default is N: ")
        if ans.lower() != "y":
            print "quit."
            exit(-1)
    prefix = object
    marker = ''
    delimiter = ''
    maxkeys = '1000'
    if options.marker:
        marker = options.marker
    if options.delimiter:
        delimiter = options.delimiter
    if options.maxkeys:
        maxkeys = options.maxkeys
    debug = True
    if not delete_all_objects(get_oss(), bucket, prefix, delimiter, marker, maxkeys, debug):
        exit(-1)

def cmd_getallbucket(args, options):
    width = 20
    print "%s %s %s" % ("CreateTime".ljust(width), "BucketLocation".ljust(width), "BucketName".ljust(width))
    marker = ""
    prefix = ""
    headers = None
    total_num = 0
    while 1:
        res = get_oss().get_service(headers, prefix, marker)
        if (res.status / 100) == 2:
            body = res.read()
            (bucket_meta_list, marker) = get_bucket_meta_list_marker_from_xml(body)
            for i in bucket_meta_list:
                print "%s %s %s" % (str(convert_to_localtime(i.creation_date)).ljust(width), i.location.ljust(width), i.name)
                total_num += 1
        else:
            break
        if not marker:
            break
    print "\nBucket Number is: %s" % total_num
    return res

def cmd_createbucket(args, options):
    check_args(2, args)
    if options.acl is not None and options.acl not in ACL_LIST:
        print "acl invalid, SHOULD be one of %s" % (ACL_LIST)
        sys.exit(1)
    acl = ''
    if options.acl:
        acl = options.acl
    bucket = parse_bucket(args[1])
    if options.location is not None:
        location = options.location
        return get_oss().put_bucket_with_location(bucket, acl, location)
    return get_oss().put_bucket(bucket, acl)

def cmd_getbucketlocation(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_bucket_location(bucket)
    if res.status / 100 == 2:
        body = res.read()
        h = GetBucketLocationXml(body)
        print h.location
    return res

def cmd_deletebucket(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    return get_oss().delete_bucket(bucket)

def cmd_deletewholebucket(args, options):
    check_args(2, args)
    ans = raw_input("DELETE whole bucket? Y/N, default is N: ")
    if ans.lower() != "y":
        print "quit."
        exit(-1)
    bucket = parse_bucket(args[1])
    debug = True
    delete_marker = ""
    delete_upload_id_marker = ""
    if options.marker:
        delete_marker = options.marker
    if options.upload_id:
        delete_upload_id_marker = options.upload_id
    if not clear_all_objects_in_bucket(get_oss(), bucket, delete_marker, delete_upload_id_marker, debug):
        exit(-1)

def delete_object(bucket, object, retry_times=2):
    object = smart_code(object)
    global DELETE_OK
    ret = False
    for i in xrange(retry_times):
        try:
            oss = get_oss()
            res = oss.delete_object(bucket, object)
            if 2 == res.status / 100:
                ret = True
            if ret:
                DELETE_OK += 1
                print "delete %s/%s OK" % (bucket, object)
                return ret
            else:
                print "delete %s/%s FAIL, status:%s, request-id:%s" % (bucket, object, res.status, res.getheader("x-oss-request-id"))
        except:
            print "delete %s/%s exception" % (bucket, object)
            print sys.exc_info()[0], sys.exc_info()[1]
    return False

class DeleteObjectWorker(threading.Thread):
    def __init__(self, retry_times, queue):
        threading.Thread.__init__(self)
        self.queue = queue
        self.retry_times = retry_times

    def run(self):
        while 1:
            try:
                (delete_object, bucket, object) = self.queue.get(block=False)
                delete_object(bucket, object, self.retry_times)
                self.queue.task_done()
            except Queue.Empty:
                break
            except:
                self.queue.task_done()

def cmd_deletebyfile(args, options):
    check_args(2, args)
    localfile = args[1]
    check_localfile(localfile)
    queue = Queue.Queue(0)
    f = open(localfile)
    for line in f:
        line = line.strip()
        (bucket, object) = parse_bucket_object(line)
        if len(bucket) != 0 and len(object) != 0:
            queue.put((delete_object, bucket, object))
    f.close()
    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES
    if options.retry_times:
        retry_times = (int)(options.retry_times)
    thread_pool = []
    for i in xrange(thread_num):
        current = DeleteObjectWorker(retry_times, queue)
        thread_pool.append(current)
        current.start()
    queue.join()
    for item in thread_pool:
        item.join()

def cmd_setacl(args, options):
    check_args(2, args)
    if options.acl is None or options.acl not in ACL_LIST:
        print "acl invalid, SHOULD be one of %s" % (ACL_LIST)
        sys.exit(1)
    bucket = parse_bucket(args[1])
    return get_oss().put_bucket(bucket, options.acl)

def cmd_getacl(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_bucket_acl(bucket)
    if (res.status / 100) == 2:
        body = res.read()
        h = GetBucketAclXml(body)
        print h.grant
    return res

def to_http_headers(string):
    headers_map = {}
    for i in string.split('#'):
        key_value_list = i.strip().split(':')
        if len(key_value_list) >= 2:
            headers_map[key_value_list[0]] = ":".join(key_value_list[1:])
    return headers_map
        
def cmd_mkdir(args, options):
    check_args(2, args)
    if not args[1].endswith('/'):
        args[1] += '/'
    (bucket, object) = parse_bucket_object(args[1])
    res = get_oss().put_object_from_string(bucket, object, "")
    return res

def handler(signum, frame):
    print 'Signal handler called with signal', signum
    raise Exception("timeout")
try:
    signal.signal(signal.SIGALRM, handler)
except:
    pass

def cmd_put(args, options):
    check_args(3, args)
    localfile = args[1]
    check_localfile(localfile)
    if os.path.getsize(localfile) > MAX_OBJECT_SIZE:
        print "locafile:%s is bigger than %s, it is not support by put, please use multiupload instead." % (localfile, MAX_OBJECT_SIZE) 
        exit(-1)
    #user specified objectname oss://bucket/[path]/object
    (bucket, object) = parse_bucket_object(args[2])
    if len(object) == 0:
        # e.g. upload to oss://bucket/
        object = os.path.basename(localfile)
    elif object.endswith("/"):
        #e.g. uplod to oss://bucket/a/b/
        object += os.path.basename(localfile)
    content_type = ""
    headers = {}
    if options.content_type:
        content_type = options.content_type
    if options.headers:
        headers = to_http_headers(options.headers)
    if options.check_md5:
        if options.check_md5.lower() == "true":
            md5string, base64md5 = get_file_md5(localfile)
            headers["Content-MD5"] = base64md5
            headers["x-oss-meta-md5"] = md5string
    timeout = 0
    if options.timeout:
        timeout = (int)(options.timeout)
        print "timeout", timeout

    try:
        signal.alarm(timeout)
    except:
        pass
    res = get_oss().put_object_from_file(bucket, object, localfile, content_type, headers)
    try:
        signal.alarm(0) # Disable the signal
    except:
        pass

    if res.status == 200:
        print_url(OSS_HOST, bucket, object, res)
    return res

def print_url(host, bucket, object, res):
    print ""
    second_level_domain = OSS_HOST
    orginal_object = object
    object = oss_quote(object)
    if check_bucket_valid(bucket) and not is_ip(second_level_domain):
        if is_oss_host(second_level_domain):
            print "Object URL is: http://%s.%s/%s" % (bucket, second_level_domain, object)
        else:
            print "Object URL is: http://%s/%s" % (second_level_domain, object)
    else:
        print "Object URL is: http://%s/%s/%s" % (second_level_domain, bucket, object)
    print "Object abstract path is: oss://%s/%s" % (bucket, orginal_object)
    header_map = convert_header2map(res.getheaders())
    print "ETag is %s " % safe_get_element("etag", header_map) 
    
def cmd_upload(args, options):
    check_args(3, args)
    localfile = args[1]
    check_localfile(localfile)
    multipart_threshold = 100*1024*1024
    if options.multipart_threshold:
        multipart_threshold = (int)(options.multipart_threshold)

    localfile_size = os.path.getsize(localfile)
    if localfile_size > multipart_threshold or localfile_size > MAX_OBJECT_SIZE:
        return cmd_multi_upload(args, options)
    return cmd_put(args, options)

def cmd_upload_group(args, options):
    check_args(3, args)
    localfile = args[1]
    check_localfile(localfile)
    #user specified objectname oss://bucket/[path]/object
    (bucket, object) = parse_bucket_object(args[2])
    if len(object) == 0:
        # e.g. upload to oss://bucket/
        object = os.path.basename(localfile)
    elif object.endswith("/"):
        #e.g. uplod to oss://bucket/a/b/
        object += os.path.basename(localfile)
    headers = {}
    content_type = ''
    if options.headers:
        headers = to_http_headers(options.headers)
    if options.content_type:
        content_type = options.content_type
        headers['Content-Type'] = content_type
    thread_num = 10
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    max_part_num = 1000
    if options.max_part_num:
        max_part_num = (int)(options.max_part_num)
    retry_times = MAX_RETRY_TIMES
    if options.retry_times:
        retry_times = (int)(options.retry_times)
    oss = get_oss()
    oss.set_retry_times(retry_times)
    res = oss.upload_large_file(bucket, object, localfile, thread_num, max_part_num, headers)

    if res.status == 200:
        print_url(OSS_HOST, bucket, object, res)
    return res

def cmd_multi_upload(args, options):
    check_args(3, args)
    localfile = args[1]
    check_localfile(localfile)
    #user specified objectname oss://bucket/[path]/object
    (bucket, object) = parse_bucket_object(args[2])
    is_check_md5 = False
    if len(object) == 0:
        # e.g. upload to oss://bucket/
        object = os.path.basename(localfile)
    elif object.endswith("/"):
        #e.g. uplod to oss://bucket/a/b/
        object += os.path.basename(localfile)
    headers = {}
    if options.headers:
        headers = to_http_headers(options.headers)
    thread_num = 10 
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    max_part_num = 1000
    if options.max_part_num:
        max_part_num = (int)(options.max_part_num)
    retry_times = MAX_RETRY_TIMES
    if options.retry_times:
        retry_times = (int)(options.retry_times)
    if options.check_md5:
        if options.check_md5.lower() == "true":
            is_check_md5 = True
            md5string, base64md5 = get_file_md5(localfile)
            headers["x-oss-meta-md5"] = md5string
    oss = get_oss()
    oss.set_retry_times(retry_times)
    upload_id = ""
    if options.upload_id:
        upload_id = options.upload_id
        res = oss.get_all_parts(bucket, object, upload_id, max_parts=1)
        if res.status != 200:
            return res

    if not upload_id:
        upload_ids = []
        upload_ids = get_upload_id_list(oss, bucket, object) 
        if upload_ids:
            upload_ids = sorted(upload_ids)
            upload_id = upload_ids[0]
        
    res = oss.multi_upload_file(bucket, object, localfile, upload_id, thread_num, max_part_num, headers, debug=True, check_md5=is_check_md5)
    if res.status == 200:
        print_url(OSS_HOST, bucket, object, res)
    return res

def cmd_copy(args, options):
    check_args(3, args)
    (bucket_source, object_source) = parse_bucket_object(args[1])
    check_bucket_object(bucket_source, object_source)
    (bucket, object) = parse_bucket_object(args[2])
    check_bucket_object(bucket, object)

    content_type = ""
    headers = {}
    if options.headers:
        headers = to_http_headers(options.headers)
    if options.content_type:
        content_type = options.content_type
        headers['Content-Type'] = content_type
    res = get_oss().copy_object(bucket_source, object_source, bucket, object, headers)
    if res.status == 200:
        print_url(OSS_HOST, bucket, object, res)
    return res

def cmd_upload_part_copy(args, options):
    check_args(3, args)
    (bucket_source, object_source) = parse_bucket_object(args[1])
    check_bucket_object(bucket_source, object_source)
    (bucket, object) = parse_bucket_object(args[2])
    check_bucket_object(bucket, object)

    #head object to get object size
    headers = {}
    res = get_oss().head_object(bucket_source, object_source, headers = headers)
    if res.status != 200:
        print 'copy large file fail because head object fail, status:%s' %(res.status)
        sys.exit(-1)
    content_len = (int)(res.getheader('Content-Length'))
    etag = res.getheader('ETag')
    #get part size
    default_part_size = 10 * 1024 * 1024
    part_size = default_part_size
    max_part_num=10000
    min_part_size = 5 * 1024 * 1024 
    if options.part_size:
        part_size = (int)(eval(options.part_size))
    if part_size < min_part_size:
        print 'part size too small, change part size to %s' %(default_part_size)
        part_size = default_part_size
    if part_size * max_part_num < content_len:
        part_size = (content_len + max_part_num - content_len % max_part_num) / max_part_num
        print 'part num more than max part num %s, change part size to %s' %(max_part_num, part_size)

    if content_len % part_size:
        part_size_list = [part_size] * (content_len / part_size) + [ content_len % part_size]
    else:
        part_size_list = [part_size] * (content_len / part_size)

    #get upload id
    if options.upload_id:
        upload_id = options.upload_id
    else:
        res = get_oss().init_multi_upload(bucket, object)
        if res.status != 200:
            print 'copy large file fail because init multipart upload fail, status:%s' %(res.status)
            sys.exit(-1)
        upload_id = GetInitUploadIdXml(res.read()).upload_id

    #upload part copy
    start = 0
    part_number = 1
    for part_size in part_size_list:
        headers = {'x-oss-copy-source-range': ('bytes=%d-%d' % (start, start + part_size-1))}
        headers['x-oss-copy-source-if-match'] = etag
        res = get_oss().copy_object_as_part(bucket_source, object_source, bucket, object, upload_id, part_number, headers)
        if res.status != 200:
            print 'copy large file fail because upload part copy fail, status:%s, upload_id:%s' %(res.status, upload_id)
            sys.exit(-1)
        start += part_size
        part_number += 1

    #complete multipart upload
    part_xml = get_part_xml(get_oss(), bucket, object, upload_id)
    res = get_oss().complete_upload(bucket, object, upload_id, part_xml)
    if res.status != 200:
        print 'copy large file fail because complete multipart upload fail, status:%s, upload_id:%s' %(res.status, upload_id)
        sys.exit(-1)
    else:
        print_url(OSS_HOST, bucket, object, res)
    return res

def copy_object(src_bucket, src_object, des_bucket, des_object, headers, replace, retry_times = 3):
    global COPY_OK
    if COPY_OK > 0 and COPY_OK % 100 == 0:
        print "%s objects are copied OK, marker is:%s" % (COPY_OK, src_object)
    for i in xrange(retry_times):
        tmp_headers = headers.copy()
        try:
            if replace:
                res = get_oss().copy_object(src_bucket, src_object, des_bucket, des_object, tmp_headers)
                if res.status == 200:
                    COPY_OK += 1
                    return True 
                else:
                    print "copy /%s/%s to /%s/%s FAIL, status:%s, request-id:%s" % \
                    (src_bucket, src_object, des_bucket, des_object, res.status, res.getheader("x-oss-request-id"))
            else:
                res = get_oss().head_object(des_bucket, des_object)
                if res.status == 200:
                    COPY_OK += 1
                    return True
                elif res.status == 404:
                    res = get_oss().copy_object(src_bucket, src_object, des_bucket, des_object, tmp_headers)
                    if res.status == 200:
                        COPY_OK += 1
                        return True
                    else:
                        print "copy /%s/%s to /%s/%s FAIL, status:%s, request-id:%s" % \
                        (src_bucket, src_object, des_bucket, des_object, res.status, res.getheader("x-oss-request-id"))
        except:
            print "copy /%s/%s to /%s/%s exception" % (src_bucket, src_object, des_bucket, des_object)
            print sys.exc_info()[0], sys.exc_info()[1]
            try:
                res = get_oss().head_object(src_bucket, src_object)
                if res.status == 200:
                    length = (int)(res.getheader('content-length'))
                    max_length = 1*1024*1024*1024
                    if length > max_length:
                        print "/%s/%s is bigger than %s, copy may fail. skip this one." \
                              % (src_bucket, src_object, max_length)
                        print "please use get command to download the object and then use multiupload command to upload the object."
                        return False
            except:
                print sys.exc_info()[0], sys.exc_info()[1]
                pass
            sleep_time = 300
            print "sleep %s" % sleep_time
            time.sleep(sleep_time)
    print "copy /%s/%s to /%s/%s FAIL" % (src_bucket, src_object, des_bucket, des_object)
    return False

class CopyObjectWorker(threading.Thread):
    def __init__(self, retry_times, queue):
        threading.Thread.__init__(self)
        self.queue = queue
        self.retry_times = retry_times

    def run(self):
        while 1:
            try:
                (copy_object, src_bucket, src_object, des_bucket, des_object, replace, headers) = self.queue.get(block=False)
                copy_object(src_bucket, src_object, des_bucket, des_object, headers, replace, self.retry_times)
                self.queue.task_done()
            except Queue.Empty:
                break
            except:
                self.queue.task_done()

def cmd_copy_bucket(args, options):
    check_args(3, args)  
    (src_bucket, src_prefix) = parse_bucket_object(args[1])
    (des_bucket, des_prefix) = parse_bucket_object(args[2])
    if des_prefix and not des_prefix.endswith("/"):
        des_prefix = "%s/" % des_prefix

    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES 
    if options.retry_times:
        retry_times = (int)(options.retry_times)
    replace = False
    if options.replace is not None and options.replace.lower() == "true":
        replace = True
    marker = ""
    if options.marker:
        marker = options.marker
    headers = {}
    if options.headers:
        headers = to_http_headers(options.headers)
    delimiter = ''
    maxkeys = '1000'
    handled_obj_num = 0
    while 1:
        queue = Queue.Queue(0)
        res = get_oss().get_bucket(src_bucket, src_prefix, marker, delimiter, maxkeys)
        if res.status != 200:
            return res
        body = res.read()
        (tmp_object_list, marker) = get_object_list_marker_from_xml(body)
        for i in tmp_object_list:
            object = i[0]
            length = i[1]
            last_modify_time = i[2]
            if str(length) == "0" and object.endswith("/"):
                continue
            handled_obj_num += 1 
            src_object = smart_code(object)
            tmp_object = src_object 
            if src_prefix.endswith("/"):
                if src_prefix == object[:len(src_prefix)]:
                    tmp_object = object[len(src_prefix):]
                while 1:
                    if not tmp_object.startswith("/"):
                        break
                    tmp_object = tmp_object[1:]
            if des_prefix:
                des_object = "%s%s" % (des_prefix, tmp_object)
            else:
                des_object = tmp_object
            queue.put((copy_object, src_bucket, src_object, des_bucket, des_object, replace, headers))
            #copy_object(src_bucket, src_object, des_bucket, des_object, replace)
        thread_pool = []
        for i in xrange(thread_num):
            current = CopyObjectWorker(retry_times, queue)
            thread_pool.append(current)
            current.start()
        queue.join()
        for item in thread_pool:
            item.join()
        if len(marker) == 0:
            break
    print "Total being copied objects num: %s, from /%s/%s to /%s/%s" % \
          (handled_obj_num, src_bucket, src_prefix, des_bucket, des_prefix)
    global COPY_OK
    print "OK num:%s" % COPY_OK
    print "FAIL num:%s" % (handled_obj_num - COPY_OK)

def continue_get(bucket, object, localfile, headers=None, retry_times=3):
    length = -1
    local_length = -2
    tmp_headers = {}
    header_map = {}
    if headers:
        tmp_headers = headers.copy()
    try:
        res = get_oss().head_object(bucket, object, tmp_headers)
        if 200 == res.status:
            length = (int)(res.getheader('content-length'))
            header_map = convert_header2map(res.getheaders())
        else:
            print "can not get the length of object:", object
            return False
    except:
        print sys.exc_info()[0], sys.exc_info()[1]
        return False
    endpos = length - 1
    for i in xrange(retry_times):
        curpos = 0
        range_info = 'bytes=%d-%d' % (curpos, endpos)
        if os.path.isfile(localfile):
            local_length = os.path.getsize(localfile)
            if i == 0 and header_map.has_key('x-oss-meta-md5'):
                oss_md5_string = header_map['x-oss-meta-md5']
                local_md5_string, base64_md5 = get_file_md5(localfile)
                if local_md5_string.lower() == oss_md5_string.lower():
                    return True
                else:
                    os.remove(localfile)
            elif local_length == length:
                #print "localfile:%s exists and length is equal. please check if it is ok. you can remove it first and download again." % localfile
                return True
            elif local_length < length:
                if i == 0:
                    os.remove(localfile)
                else:
                    curpos = local_length
                    range_info = 'bytes=%d-%d' % (curpos, endpos)
                    print "localfile:%s exists and length is:%s, continue to download. range:%s." % (localfile, local_length, range_info)
            else:
                os.remove(localfile)
        file = open(localfile, "ab+")
        tmp_headers = {}
        if headers:
            tmp_headers = headers.copy()
        tmp_headers['Range'] = range_info
        file.seek(curpos)
        is_read_ok = False
        oss_md5_string = ''
        try:
            res = get_oss().get_object(bucket, object, tmp_headers)
            if res.status/100 == 2:
                header_map = convert_header2map(res.getheaders())
                if header_map.has_key('x-oss-meta-md5'):
                    oss_md5_string = header_map['x-oss-meta-md5']
                while True:
                    content = res.read(RECV_BUF_SIZE)
                    if content:
                        file.write(content)
                        curpos += len(content)
                    else:
                        break
                is_read_ok = True
            else:
                print "range get /%s/%s [%s] ret:%s, request-id:%s" % (bucket, object, range_info, res.status, res.getheader("x-oss-request-id"))
        except:
            print "range get /%s/%s [%s] exception" % (bucket, object, range_info)
            print sys.exc_info()[0], sys.exc_info()[1]
            file.flush()
            file.close()
            file_opened = False
            continue

        file.flush()
        file.close()

        if os.path.isfile(localfile):
            local_length = os.path.getsize(localfile)
        if is_read_ok and length == local_length:
            if oss_md5_string != '':
                md5string, base64md5 = get_file_md5(localfile)
                if md5string.lower() != oss_md5_string.lower():
                    print "The object %s is download to %s failed. file md5 is incorrect." % (object, localfile)
                    return False
            return True
        else:
            print "The object %s is download to %s failed. file length is incorrect.length is:%s local_length:%s" % (object, localfile, length, local_length)
    return False

def cmd_get(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    localfile = args[2]
    localfile = smart_code(localfile)
    headers = {}
    if options.headers:
        headers = to_http_headers(options.headers)
    if options.continue_download:
        retry_times = 3
        res = continue_get(bucket, object, localfile, headers, retry_times)
    else:
        tmp_headers = {}
        tmp_headers = headers.copy()
        res = get_oss().get_object_to_file(bucket, object, localfile, headers=tmp_headers)
        if res.status/100 == 2:
            header_map = convert_header2map(res.getheaders())
            if header_map.has_key('x-oss-meta-md5'):
                oss_md5string = header_map['x-oss-meta-md5']
                md5string, base64md5 = get_file_md5(localfile)
                if md5string.lower() != oss_md5string.lower():
                    print "The object %s is download to %s failed. file md5 is incorrect." % (object, localfile)
                    sys.exit(1)
            else:
                content_length = int(header_map['content-length'])
                local_length = os.path.getsize(localfile)
                if content_length != local_length:
                    print "The object %s is download to %s failed. file length is incorrect." % (object, localfile)
                    sys.exit(1)
        else:
            return res
    if res:
        print "The object %s is downloaded to %s, please check." % (object, localfile)
    return res

def cmd_multi_get(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    localfile = args[2]
    localfile = smart_code(localfile)
    thread_num = 5
    if options.thread_num:
        thread_num = (int)(options.thread_num)
    retry_times = MAX_RETRY_TIMES
    if options.retry_times:
        retry_times = (int)(options.retry_times)

    show_bar = False
    oss = get_oss(show_bar)
    ret = multi_get(oss, bucket, object, localfile, thread_num, retry_times)
    if ret:
        print "The object %s is downloaded to %s, please check." % (object, localfile)
    else:
        print "Download object:%s failed!" % (object)
        exit(-1)

def cmd_cat(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    res = get_oss().get_object(bucket, object)
    if res.status == 200:
        data = ""
        while 1:
            data = res.read(10240)
            if len(data) != 0:
                print data
            else:
                break
    return res

def cmd_meta(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    headers = {}
    res = get_oss().head_object(bucket, object, headers = headers)
    if res.status == 200:
        header_map = convert_header2map(res.getheaders())
        width = 16
        print "%s: %s" % ("objectname".ljust(width), object)
        for key, value in header_map.items():
            print "%s: %s" % (key.ljust(width), value)
    return res

def cmd_info(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    res = get_oss().get_object_info(bucket, object)
    if res.status == 200:
        print res.read()
    return res

def cmd_delete(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    return get_oss().delete_object(bucket, object)

def cmd_cancel(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    if options.upload_id is None:
        print "upload_id invalid, please set with --upload_id=xxx"
        sys.exit(1)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    return get_oss().cancel_upload(bucket, object, options.upload_id)

def cmd_sign_url(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    if options.timeout:
        timeout = options.timeout
    else:
        timeout = "600"
    print "timeout is %s seconds." % timeout
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    method = 'GET'
    print get_oss().sign_url(method, bucket, object, int(timeout))

def cmd_configure(args, options):
    if options.accessid is None or options.accesskey is None:
        print "%s miss parameters, use --accessid=[accessid] --accesskey=[accesskey] to specify id/key pair" % args[0]
        sys.exit(-1) 
    config = ConfigParser.RawConfigParser()
    config.add_section(CONFIGSECTION)
    if options.host is not None:
        config.set(CONFIGSECTION, 'host', options.host)
    config.set(CONFIGSECTION, 'accessid', options.accessid)
    config.set(CONFIGSECTION, 'accesskey', options.accesskey)
    if options.sts_token:
        config.set(CONFIGSECTION, 'sts_token', options.sts_token)
    cfgfile = open(CONFIGFILE, 'w+')
    config.write(cfgfile)
    print "Your configuration is saved into %s ." % CONFIGFILE
    cfgfile.close()
    import stat
    os.chmod(CONFIGFILE, stat.S_IREAD | stat.S_IWRITE)

def cmd_help(args, options):
    print HELP 

def cmd_create_link(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    object_list = args[2:] 
    return get_oss().create_link_from_list(bucket, object, object_list)

def cmd_create_link_from_file(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    local_file = args[2]
    if not os.path.isfile(local_file):
        print "no such file:%s" % local_file
        exit(-1)
    f = open(local_file)
    object_list = f.readlines()
    f.close()
    return get_oss().create_link_from_list(bucket, object, object_list)

def cmd_get_link_index(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    res = get_oss().get_link_index(bucket, object)
    if res.status == 200:
        print res.read()
    return res

def cmd_create_group_from_file(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    local_file = args[2]
    if not os.path.isfile(local_file):
        print "no such file:%s" % local_file
        exit(-1)
    f = open(local_file)
    object_list = f.readlines()
    f.close()
    part_msg_list = []
    for i in range(len(object_list)):
        object_list[i] = object_list[i].rstrip('\n')
        res = get_oss().head_object(bucket, object_list[i])
        if res.status != 200:
            print "head object: ", object_list[i], ", ", res.status
            print 'Create Group Fail!' 
            return res
        header_map = convert_header2map(res.getheaders())
        etag = safe_get_element("etag", header_map)
        etag = etag.replace("\"", "")
        list = [str(i), object_list[i], etag]
        part_msg_list.append(list)
    object_group_msg_xml = create_object_group_msg_xml(part_msg_list)
    return get_oss().post_object_group(bucket, object, object_group_msg_xml)

def cmd_create_group(args, options):
    check_args(3, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    object_list = args[2:] 
    part_msg_list = []
    for i in range(len(object_list)):
        res = get_oss().head_object(bucket, object_list[i])
        if res.status != 200:
            print "head object: ", object_list[i], ", ", res.status
            print 'Create Group Fail!' 
            return res
        header_map = convert_header2map(res.getheaders())
        etag = safe_get_element("etag", header_map)
        etag = etag.replace("\"", "")
        list = [str(i), object_list[i], etag]
        part_msg_list.append(list)
    object_group_msg_xml = create_object_group_msg_xml(part_msg_list)
    return get_oss().post_object_group(bucket, object, object_group_msg_xml)

def cmd_get_group_index(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1]) 
    check_bucket_object(bucket, object)
    res = get_oss().get_object_group_index(bucket, object)
    if res.status == 200:
        print res.read()
    return res

def cmd_put_bucket_logging(args, options):
    source_bucket = ''
    target_bucket = ''
    prefix = ''
    check_args(2, args)
    if len(args) >= 3:
        target_bucket = args[2]
        (target_bucket, prefix) = parse_bucket_object(args[2])
    source_bucket = parse_bucket(args[1])
    target_bucket = parse_bucket(args[2])
    res = get_oss().put_logging(source_bucket, target_bucket, prefix)
    return res

def cmd_get_bucket_logging(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_logging(bucket)
    if res.status == 200:
        print res.read()
    return res

def cmd_put_bucket_website(args, options):
    bucket = ''
    indexfile = ''
    errorfile = ''
    check_args(3, args)
    if len(args) >= 3:
        indexfile = args[2]
    if len(args) >= 4:
        errorfile = args[3]
    bucket = parse_bucket(args[1])
    res = get_oss().put_website(bucket, indexfile, errorfile)
    return res

def cmd_get_bucket_website(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_website(bucket)
    if res.status == 200:
        print res.read()
    return res

def cmd_delete_bucket_website(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().delete_website(bucket)
    return res

def cmd_delete_bucket_logging(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().delete_logging(bucket)
    return res

def cmd_put_bucket_cors(args, options):
    check_args(3, args)
    bucket = parse_bucket(args[1])
    local_file = args[2]
    if not os.path.isfile(local_file):
        print "no such file:%s" % local_file
        exit(-1)
    f = open(local_file)
    content = f.read()
    f.close()
    return get_oss().put_cors(bucket, content)

def cmd_get_bucket_cors(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_cors(bucket)
    if res.status == 200:
        print res.read()
    return res

def cmd_delete_bucket_cors(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().delete_cors(bucket)
    return res

def cmd_options(args, options):
    check_args(2, args)
    (bucket, object) = parse_bucket_object(args[1])
    headers = {}
    is_ok = True
    if options.origin:
        headers['Origin'] = options.origin
    else:
        is_ok = False
    method_list = ["GET", "PUT", "DELETE", "HEAD", "POST"]
    if options.method:
        if options.method not in method_list:
            is_ok = False
        else:
            headers['Access-Control-Request-Method'] = options.method
    else:
        is_ok = False
    if not is_ok:
        print "please set origin and method with --origin=xxx --method=xxx, the value of --method SHOULD be one of %s" % (" ".join(method_list))
        exit(-1)
    res = get_oss().options(bucket, object, headers)
    return res

def cmd_put_bucket_lifecycle(args, options):
    check_args(3, args)
    bucket = parse_bucket(args[1])
    local_file = args[2]
    if not os.path.isfile(local_file):
        print "no such file:%s" % local_file
        exit(-1)
    f = open(local_file)
    lifecycle_config = f.read()
    f.close()
    res = get_oss().put_lifecycle(bucket, lifecycle_config)
    return res

def cmd_get_bucket_lifecycle(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_lifecycle(bucket)
    if res.status == 200:
        print res.read()
    return res

def cmd_put_bucket_referer(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    allow_empty_referer = True
    if options.allow_empty_referer and options.allow_empty_referer.lower() == "false":
        allow_empty_referer = False
    referer_list = []
    if options.referer:
        referer_list = options.referer.split(",")
    res = get_oss().put_referer(bucket, allow_empty_referer, referer_list)
    return res

def cmd_get_bucket_referer(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().get_referer(bucket)
    if res.status == 200:
        print res.read()
    return res

def cmd_delete_bucket_lifecycle(args, options):
    check_args(2, args)
    bucket = parse_bucket(args[1])
    res = get_oss().delete_lifecycle(bucket)
    return res

def get_oss(show_bar = True):
    oss = OssAPI(OSS_HOST, ID, KEY, sts_token=STS_TOKEN)
    oss.show_bar = show_bar 
    oss.set_send_buf_size(SEND_BUF_SIZE)
    oss.set_recv_buf_size(RECV_BUF_SIZE)
    oss.set_debug(IS_DEBUG)
    return oss

def setup_credentials(options):
    config = ConfigParser.ConfigParser()
    try:
        config.read(CONFIGFILE)
        global OSS_HOST
        global ID
        global KEY
        global STS_TOKEN
        try:
            OSS_HOST = config.get(CONFIGSECTION, 'host')
        except Exception:
            OSS_HOST = DEFAUL_HOST 
        ID = config.get(CONFIGSECTION, 'accessid')
        KEY = config.get(CONFIGSECTION, 'accesskey')
        try:
            STS_TOKEN = config.get(CONFIGSECTION, 'sts_token')
        except:
            pass
        if options.accessid is not None:
            ID = options.accessid
        if options.accesskey is not None:
            KEY = options.accesskey
        if options.sts_token is not None:
            STS_TOKEN = options.sts_token
        if options.host is not None:
            OSS_HOST = options.host
    except Exception:
        if options.accessid is not None:
            ID = options.accessid
        if options.accesskey is not None:
            KEY = options.accesskey
        if options.sts_token is not None:
            STS_TOKEN = options.sts_token
        if options.host is not None:
            OSS_HOST = options.host

        if len(ID) == 0 or len(KEY) == 0:
            print "can't get accessid/accesskey, setup use : config --accessid=accessid --accesskey=accesskey"
            sys.exit(1)

def setup_cmdlist():

    CMD_LIST['GetAllBucket'] = cmd_getallbucket
    CMD_LIST['CreateBucket'] = cmd_createbucket
    CMD_LIST['DeleteBucket'] = cmd_deletebucket
    CMD_LIST['DeleteWholeBucket'] = cmd_deletewholebucket
    CMD_LIST['DeleteByFile'] = cmd_deletebyfile
    CMD_LIST['GetBucketLocation'] = cmd_getbucketlocation

    CMD_LIST['GetAcl'] = cmd_getacl
    CMD_LIST['SetAcl'] = cmd_setacl


    CMD_LIST['List'] = cmd_listing
    CMD_LIST['MkDir'] = cmd_mkdir
    CMD_LIST['Init'] = cmd_init_upload
    CMD_LIST['UploadPartFromString'] = cmd_upload_part_from_string

    CMD_LIST['UploadPartFromFile'] = cmd_upload_part_from_file
    CMD_LIST['ListPart'] = cmd_listpart
    CMD_LIST['ListParts'] = cmd_listparts
    CMD_LIST['GetAllPartSize'] = cmd_getallpartsize
    CMD_LIST['ListAllObject'] = cmd_listallobject
    CMD_LIST['ListAllDir'] = cmd_listalldir
    CMD_LIST['DownloadAllObject'] = cmd_downloadallobject
    CMD_LIST['UploadObjectFromLocalDir'] = cmd_upload_object_from_localdir
    CMD_LIST['UploadDisk'] = cmd_upload_disk
    CMD_LIST['DeleteAllObject'] = cmd_deleteallobject
    CMD_LIST['Put'] = cmd_put
    CMD_LIST['Copy'] = cmd_copy
    CMD_LIST['CopyLargeFile'] = cmd_upload_part_copy
    CMD_LIST['CopyBucket'] = cmd_copy_bucket
    CMD_LIST['Upload'] = cmd_upload
    CMD_LIST['UploadGroup'] = cmd_upload_group
    CMD_LIST['MultiUpload'] = cmd_multi_upload

    CMD_LIST['Get'] = cmd_get

    CMD_LIST['MultiGet'] = cmd_multi_get
    CMD_LIST['Cat'] = cmd_cat
    CMD_LIST['Meta'] = cmd_meta
    CMD_LIST['Info'] = cmd_info

    CMD_LIST['Delete'] = cmd_delete

    CMD_LIST['Cancel'] = cmd_cancel

    CMD_LIST['Config'] = cmd_configure
    CMD_LIST['Help'] = cmd_help
    CMD_LIST['SignUrl'] = cmd_sign_url

    CMD_LIST['CreateLink'] = cmd_create_link
    CMD_LIST['CreateLinkFromFile'] = cmd_create_link_from_file
    CMD_LIST['GetLinkIndex'] = cmd_get_link_index

    CMD_LIST['CreateGroup'] = cmd_create_group
    CMD_LIST['CreateGroupFromFile'] = cmd_create_group_from_file
    CMD_LIST['GetGroupIndex'] = cmd_get_group_index

    CMD_LIST['PutBucketLogging'] = cmd_put_bucket_logging
    CMD_LIST['GetBucketLogging'] = cmd_get_bucket_logging
    CMD_LIST['DeleteBucketLogging'] = cmd_delete_bucket_logging
    CMD_LIST['PutBucketWebsite'] = cmd_put_bucket_website
    CMD_LIST['GetBucketWebsite'] = cmd_get_bucket_website
    CMD_LIST['DeleteBucketWebsite'] = cmd_delete_bucket_website

    CMD_LIST['PutBucketCors'] = cmd_put_bucket_cors
    CMD_LIST['GetBucketCors'] = cmd_get_bucket_cors
    CMD_LIST['DeleteBucketCors'] = cmd_delete_bucket_cors
    CMD_LIST['Options'] = cmd_options

    CMD_LIST['PutBucketLifeCycle'] = cmd_put_bucket_lifecycle
    CMD_LIST['GetBucketLifeCycle'] = cmd_get_bucket_lifecycle
    CMD_LIST['DeleteBucketLifeCycle'] = cmd_delete_bucket_lifecycle

    CMD_LIST['PutBucketReferer'] = cmd_put_bucket_referer
    CMD_LIST['GetBucketReferer'] = cmd_get_bucket_referer

def getSuitableKeyValues(keyValues):
    newMap = dict()
    if keyValues is not None and isinstance(keyValues,dict):
        keys = keyValues.keys()
        for key in keys:
            value = keyValues.get(key)
            if value is not None and isinstance(value,list) and len(value)>0:
                value = value[0]
                newMap[key] = value
    return newMap

def getParameterList():
    parametersList = ['origin','sts_token', 'force', 'recv_buf_size', 'accesskey', 'part_size', 'retry_times',\
                  'replace', 'thread_num', 'marker', 'exclude','skip_dir', 'out', 'check_point', 'strip_dir',\
                  'check_md5','delimiter', 'skip_suffix', 'maxkeys', 'filename_list', 'location', 'temp_dir', \
                  'method', 'config_file', 'accessid', 'continue_download', 'allow_empty_referer','host',\
                  'referer', 'content_type', 'data', 'device_id', 'max_part_num', 'acl','headers',\
                  'part_number', 'upload_id', 'send_buf_size', 'timeout', 'debug', 'multipart_threshold']
    return parametersList

def initKeyValues(parametersList):
    newMap = dict.fromkeys(parametersList)
    return newMap

def getParametersKV(keyValues,parameters):
    if isinstance(keyValues,dict) and isinstance(parameters,dict):
        keys = parameters.keys()
        for item in keyValues:
            if item in keys:
                parameters[item] = keyValues[item]
    return parameters


def getOptionsFromDict(parameters):
    if isinstance(parameters,dict):
        options = Values(parameters)
        return options

def getOperations(operation):
    list = []
    if operation is not None:
        list.append(operation)
    return list

def getAvailableOperations():
    setup_cmdlist()
    return CMD_LIST.keys()

def handleOss():
    parser = aliyunCliParser()
    operation = parser._getOperations()
    keyValues = parser._getKeyValues()
    keyValues = parser.getOpenApiKeyValues(keyValues)
    keyValues = getSuitableKeyValues(keyValues)
    parameterList = getParameterList()

    parameters = initKeyValues(parameterList)
    parameters = getParametersKV(keyValues,parameters)

    options = getOptionsFromDict(parameters)

    args = operation
    setup_cmdlist()

    if args is None or len(args) < 1 or args[0] in HELP_CMD_LIST:
        print HELP
        sys.exit(1)

    if args[0] not in CMD_LIST.keys():
        print "unsupported command : %s " % args[0]
        print HELP
        sys.exit(1)

    if options.config_file is not None:
        CONFIGFILE = options.config_file

    if options.debug is not None:
        debug = options.debug
        if debug.lower() == "true":
            IS_DEBUG = True
        else:
            IS_DEBUG = False

    if options.send_buf_size is not None:
        try:
            SEND_BUF_SIZE = (int)(options.send_buf_size)
        except ValueError:
            pass

    if options.recv_buf_size is not None:
        try:
            RECV_BUF_SIZE = (int)(options.recv_buf_size)
        except ValueError:
            pass

    if options.upload_id is not None:
        check_upload_id(options.upload_id)



    if args[0] != 'Config':
        setup_credentials(options)
    else:
        CMD_LIST['Config'](args, options)
        sys.exit(0)


    cmd = args[0]
    begin = time.time()

    try:
        res = CMD_LIST[cmd](args, options)
        print_result(cmd, res)
    except socket.timeout:
        print "Socket timeout, please try again later."
        sys.exit(1)
    except socket.error, args:
        print "Connect to oss failed: %s.\nplease check the host name you provided could be reached.\ne.g:" % (args)
        print "\tcurl %s\nor\n\tping %s\n" % (OSS_HOST, OSS_HOST)
        sys.exit(1)

    end = time.time()
    sys.stderr.write("%.3f(s) elapsed\n" % (end - begin))


if __name__ == '__main__':
    handleOss()
