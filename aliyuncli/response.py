'''
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing,
 software distributed under the License is distributed on an
 "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 KIND, either express or implied.  See the License for the
 specific language governing permissions and limitations
 under the License.
'''
# -*- coding:utf-8 -*-

import sys
import json
import text
from table import MultiTable
import jmespath


class Response(object):

    def __init__(self,args):
        self.args = args
    def __call__(self, command,response, stream=None):
        if stream is None:
            stream = sys.stdout
        if _has_filter_param(self.args)[0]:
            filter_value =_has_filter_param(self.args)[1]
            expression = jmespath.compile(filter_value)
            response = expression.search(response)
        try:
            self._format_response(command, response, stream)
        except IOError as e:
            pass
        finally:
            self._flush_stream(stream)
    def _flush_stream(self, stream):
        try:
            stream.flush()
        except IOError:
            pass


class JSONResponse(Response):
    def _format_response (self, command,response,stream=None):
        if stream is None :
            stream = sys.stdout
        if response:
            json.dump(response,stream, indent=4)
            stream.write('\n')


class TextResponse (Response):
    def __call__(self, command, response, stream=None):
        if stream is None:
            stream = sys.stdout
        try:
            self._format_response(response, stream)
        finally:
            # flush is needed to avoid the "close failed in file object
            # destructor" in python2.x (see http://bugs.python.org/issue11380).
            self._flush_stream(stream)

    def _format_response(self, response, stream):
        if _has_filter_param(self.args)[0]:
            filter_value =_has_filter_param(self.args)[1]
            expression = jmespath.compile(filter_value)
            response = expression.search(response)
        text.format_text(response, stream)

class TableResponse (Response):
    def __init__(self, args, table=None):
        super(TableResponse, self).__init__(args)
        self.table = MultiTable(initial_section=False,
                                    column_separator='|')

    def _format_response(self, command, response, stream):
        if self._build_table(command, response):
            try:
                self.table.render(stream)
            except IOError:
                # If they're piping stdout to another process which exits before
                # we're done writing all of our output, we'll get an error about a
                # closed pipe which we can safely ignore.
                pass

    def _build_table(self, title, current, indent_level=0):
        if not current:
            return False
        if title is not None:
            self.table.new_section(title, indent_level=indent_level)
        if isinstance(current, list):
            if isinstance(current[0], dict):
                self._build_sub_table_from_list(current, indent_level, title)
            else:
                for item in current:
                    if self._scalar_type(item):
                        self.table.add_row([item])
                    elif all(self._scalar_type(el) for el in item):
                        self.table.add_row(item)
                    else:
                        self._build_table(title=None, current=item)
        if isinstance(current, dict):
            # Render a single row section with keys as header
            # and the row as the values, unless the value
            # is a list.
            self._build_sub_table_from_dict(current, indent_level)
        return True

    def _build_sub_table_from_dict(self, current, indent_level):
        # Render a single row section with keys as header
        # and the row as the values, unless the value
        # is a list.
        headers, more = self._group_scalar_keys(current)
        if len(headers) == 1:
            # Special casing if a dict has a single scalar key/value pair.
            self.table.add_row([headers[0], current[headers[0]]])
        elif headers:
            self.table.add_row_header(headers)
            self.table.add_row([current[k] for k in headers])
        for remaining in more:
            self._build_table(remaining, current[remaining],indent_level=indent_level + 1)

    def _build_sub_table_from_list(self, current, indent_level, title):
        headers, more = self._group_scalar_keys_from_list(current)
        self.table.add_row_header(headers)
        first = True
        for element in current:
            if not first and more:
                self.table.new_section(title,
                                       indent_level=indent_level)
                self.table.add_row_header(headers)
            first = False
            # Use .get() to account for the fact that sometimes an element
            # may not have all the keys from the header.
            self.table.add_row([element.get(header, '') for header in headers])
            for remaining in more:
                # Some of the non scalar attributes may not necessarily
                # be in every single element of the list, so we need to
                # check this condition before recursing.
                if remaining in element:
                    self._build_table(remaining, element[remaining],
                                    indent_level=indent_level + 1)

    def _scalar_type(self, element):
        return not isinstance(element, (list, dict))

    def _group_scalar_keys_from_list(self, list_of_dicts):
        # We want to make sure we catch all the keys in the list of dicts.
        # Most of the time each list element has the same keys, but sometimes
        # a list element will have keys not defined in other elements.
        headers = set()
        more = set()
        for item in list_of_dicts:
            current_headers, current_more = self._group_scalar_keys(item)
            headers.update(current_headers)
            more.update(current_more)
        headers = list(sorted(headers))
        more = list(sorted(more))
        return headers, more

    def _group_scalar_keys(self, current):
        # Given a dict, separate the keys into those whose values are
        # scalar, and those whose values aren't.  Return two lists,
        # one is the scalar value keys, the second is the remaining keys.
        more = []
        headers = []
        for element in current:
            if self._scalar_type(current[element]):
                headers.append(element)
            else:
                more.append(element)
        headers.sort()
        more.sort()
        return headers, more


def _has_filter_param(args):
    has = False
    param =None
    if isinstance(args,dict):
        value = args.get('filter')
        if isinstance(value,list) and len(value)>0:
            param=value[0]
            param = param.strip()
            if len(param) >0:
                has=True
    return [has,param]



def get_response (output_type,parsed_args):
    if output_type == None :
        output_type = 'JSON'
    output_type = output_type.lower()
    if output_type == 'json':
            return JSONResponse(parsed_args)
    elif output_type  == 'text':
        return TextResponse(parsed_args)
    elif output_type == 'table':
        return TableResponse(parsed_args)
    raise ValueError("Unknown output type: %s" % output_type)



def display_response(command, response,output,parsed_globals=None):
    if output is None:
        output = 'JSON'
    formatter = get_response(output, parsed_globals)
    formatter(command, response)


