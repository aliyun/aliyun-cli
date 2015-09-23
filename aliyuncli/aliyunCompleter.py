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

import sys
import logging
import copy
import aliyunOpenApiData

LOG = logging.getLogger(__name__)


class Completer(object):

    def __init__(self):
        self.openApiDataHandler = aliyunOpenApiData.aliyunOpenApiDataHandler()
        self.driver = None
        self.main_hc = None
        self.main_options = ['output', 'AccessKeyId', 'AccessKeySecret', 'RegionId' ,'profile', 'version']
        self.cmdline = None
        self.point = None
        self.command_hc = None
        self.subcommand_hc = None
        self.command_name = None
        self.operation = None
        self.current_word = None
        self.previous_word = None
        self.non_options = None
        self.version = None
        self.aliyuncli = 'aliyuncli'

    def _complete_option(self, option_name):
        # if option_name == '--endpoint-url':
        #     return []
        if option_name == '--output':
            cli_data = ['text', 'table', 'json']
            return cli_data
        # if option_name == '--profile':
        #     return self.driver.session.available_profiles
        return []

    def _complete_provider(self):
        retval = []
        if self.current_word.startswith('-'):
            cw = self.current_word.lstrip('-')
            l = ['--' + n for n in self.main_options
                 if n.startswith(cw)]
            retval = l
        elif self.current_word == './testcli' or self.current_word == self.aliyuncli:
            retval = self._documented(self.openApiDataHandler.getApiCmdsLower())
        else:
            # Otherwise, see if they have entered a partial command name
            retval = self._documented(self.openApiDataHandler.getApiCmdsLower(),
                                      startswith=self.current_word)
        return retval

    def _complete_command(self):
        retval = []
        if self.current_word == self.command_name: # here means only cmd give operation is None
            _operations = set()
            apiOperations = self.openApiDataHandler.getApiOperations(self.command_name, self.version)
            import commandConfigure
            _configure = commandConfigure.commandConfigure()
            extensionOperations = _configure.getExtensionOperations(self.command_name)
            for item in apiOperations:
                _operations.add(item)
            if extensionOperations is not None:
                for item in extensionOperations:
                    _operations.add(item)
            if self.openApiDataHandler.getApiOperations(self.command_name, self.version):
                retval = self._documented(_operations)
            # retval = self._documented(self.openApiDataHandler.getApiOperations(self.command_name, self.version))
        elif self.current_word.startswith('-'): # this is complete the key and values
            retval = self._find_possible_options()
        else: # here means cmd give we need complete the operation
            # See if they have entered a partial command name
            _operations = set()
            apiOperations = self.openApiDataHandler.getApiOperations(self.command_name, self.version)
            import commandConfigure
            _configure = commandConfigure.commandConfigure()
            extensionOperations = _configure.getExtensionOperations(self.command_name)
            for item in apiOperations:
                _operations.add(item)
            if extensionOperations is not None:
                for item in extensionOperations:
                    _operations.add(item)
            if self.openApiDataHandler.getApiOperations(self.command_name, self.version):
                retval = self._documented(_operations, startswith=self.current_word)
                # retval = self._documented(self.openApiDataHandler.getApiOperations(self.command_name, self.version),
                #                           startswith=self.current_word)
        return retval

    def _documented(self, table, startswith=None):
        names = []
        for key in table:
            # if getattr(command, '_UNDOCUMENTED', False):
                # Don't tab complete undocumented commands/params
                # continue
            if startswith is not None and not key.startswith(startswith):
                continue
            # if getattr(command, 'positional_arg', False):
            #     continue
            names.append(key)
        return names

    def _complete_subcommand(self):
        retval = []
        if self.current_word == self.operation:
            retval = []
        elif self.current_word.startswith('-'):
            retval = self._find_possible_options()
        return retval

    def _find_possible_options(self):
        all_options = copy.copy(self.main_options)
        # here give all attribute list
        # where code run here , self.version should be decide before
        # self.subcommand_name = self.operation
        # cmdInstance = self.openApiDataHandler.getInstanceByCmd(self.command_name, self.operation, self.version)
        cmdInstance, mclassname = self.openApiDataHandler.getInstanceByCmdOperation(self.command_name, self.operation, self.version)
        # old_arg_list = self.openApiDataHandler.getAttrList(cmdInstance)
        old_arg_list = list()
        if cmdInstance is None:
            import commandConfigure
            _configure = commandConfigure.commandConfigure()
            old_arg_list = _configure.getExtensionOptions(self.command_name, self.operation)
        else:
            old_arg_list = self.openApiDataHandler.getAttrList(mclassname)
        new_arg_list = set()
        if not old_arg_list is None:
            for item in old_arg_list:
                if not item.startswith('_'):
                    new_arg_list.add(item)
            all_options = all_options + self._documented(new_arg_list)
        for opt in self.options:
            # Look thru list of options on cmdline. If there are
            # options that have already been specified and they are
            # not the current word, remove them from list of possibles.
            if opt != self.current_word:
                stripped_opt = opt.lstrip('-')
                if stripped_opt in all_options:
                    all_options.remove(stripped_opt)
        cw = self.current_word.lstrip('-')
        possibles = ['--' + n for n in all_options if n.startswith(cw)]
        if len(possibles) == 1 and possibles[0] == self.current_word:
            return self._complete_option(possibles[0])
        return possibles

    def _help_to_show_instance_attribute(self, classname):
        all_options = copy.copy(self.main_options)
        # here give all attribute list
        # where code run here , self.version should be decide before
        # self.subcommand_name = self.operation
        old_arg_list = self.openApiDataHandler.getAttrList(classname)
        new_arg_list = set()
        if not old_arg_list is None:
            for item in old_arg_list:
                if not item.startswith('_'):
                    new_arg_list.add(item)
            all_options = all_options + self._documented(new_arg_list)
        # for opt in self.options:
        #     # Look thru list of options on cmdline. If there are
        #     # options that have already been specified and they are
        #     # not the current word, remove them from list of possibles.
        #     if opt != self.current_word:
        #         stripped_opt = opt.lstrip('-')
        #         if stripped_opt in all_options:
        #             all_options.remove(stripped_opt)
        #cw = self.current_word.lstrip('-')
        possibles = ['--' + n for n in all_options]
        # if len(possibles) == 1 and possibles[0] == self.current_word:
        #     return self._complete_option(possibles[0])
        return possibles

    def _process_command_line(self):
        # Process the command line and try to find:
        #     - command_name
        #     - subcommand_name
        #     - words
        #     - current_word
        #     - previous_word
        #     - non_options
        #     - options
        self.command_name = None
        self.operation = None
        self.words = self.cmdline[0:self.point].split()
        self.current_word = self.words[-1]
        if len(self.words) >= 2:
            self.previous_word = self.words[-2]
        else:
            self.previous_word = None
        self.non_options = [w for w in self.words if not w.startswith('-')]
        self.options = [w for w in self.words if w.startswith('-')]
        # Look for a command name in the non_options
        for w in self.non_options:
            if w in self.openApiDataHandler.getApiCmdsLower() or w in self.openApiDataHandler.getApiCmds(): # cmd check
                self.command_name = w # here give the command_name
                if self.command_name.lower()=="oss":
                    return
                self.version = self.openApiDataHandler.getSdkVersion(self.command_name, None)
                cmd_obj = self.openApiDataHandler.getApiOperations(self.command_name, self.version)
                # self.command_hc = cmd_obj.create_help_command()
                if not cmd_obj is None:
                #     Look for subcommand name
                    for w in self.non_options:
                        if w in cmd_obj:
                            self.operation = w
                            # cmd_obj = self.command_hc.command_table[self.subcommand_name]
                            # self.subcommand_hc = cmd_obj.create_help_command()
                            break
                cmd_extension_obj = self.openApiDataHandler.getExtensionOperationsFromCmd(self.command_name)
                if not cmd_extension_obj is None:
                    for w in self.non_options:
                        if w in cmd_extension_obj:
                            self.operation = w
                            # cmd_obj = self.command_hc.command_table[self.subcommand_name]
                            # self.subcommand_hc = cmd_obj.create_help_command()
                            break
                break

    def complete(self, cmdline, point):
        self.cmdline = cmdline
        self.command_name = None
        if point is None:
            point = len(cmdline)
        self.point = point
        self._process_command_line()
        if not self.command_name: # such as 'ec'
            # If we didn't find any command names in the cmdline
            # lets try to complete provider options
            return self._complete_provider()
        if self.command_name and not self.operation: # such as 'ecs create-'
            return self._complete_command()
        return self._complete_subcommand()


def complete(cmdline, point):
    choices = Completer().complete(cmdline, point)
    print(' \n'.join(choices))


if __name__ == '__main__':
    # if len(sys.argv) == 3:
    #     cmdline = sys.argv[1]
    #     point = int(sys.argv[2])
    # elif len(sys.argv) == 2:
    #     cmdline = sys.argv[1]
    # else:
    #     print('usage: %s <cmdline> <point>' % sys.argv[0])
    #     sys.exit(1)
    cmdline = './testcli E'
    point = len(cmdline)
    print(complete(cmdline, point))
