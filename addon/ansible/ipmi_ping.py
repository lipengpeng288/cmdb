from __future__ import (absolute_import, division, print_function)
__metaclass__ = type

import subprocess

from ansible.plugins.action import ActionBase

class ActionModule(ActionBase):
    '''IPMI Ping Module for Ansible 2.3+
    '''
    _ipmi_addr = None
    _ipmi_user = None
    _ipmi_pass = None

    def run(self, tmp=None, task_vars=None):
        if task_vars is None:
            task_vars = dict()

        result = super(ActionModule, self).run(tmp, task_vars)
        del tmp # tmp no longer has any effect

        try:
            self._ipmi_addr = task_vars["ipmi_addr"]
        except KeyError:
            self._ipmi_addr = None
        try:
            self._ipmi_user = task_vars["ipmi_user"]
        except KeyError:
            self._ipmi_user = None
        try:
            self._ipmi_pass = task_vars["ipmi_pass"]
        except KeyError:
            self._ipmi_pass = None

        if not self._ipmi_addr or not self._ipmi_user or not self._ipmi_pass:
            result['failed'] = True
            result['msg'] = 'Invalid IPMI endpoint or credential'
        reason = ''
        warning = ''
        try:
            subprocess.check_output(['ipmitool', '-I', 'lanplus', '-H', self._ipmi_addr, '-U', self._ipmi_user, '-P', self._ipmi_pass, 'fru', 'print'])
        except subprocess.CalledProcessError as e:
            if e.returncode == 1 and 'FRU' in e.output:
                warning = 'Information was successfully retrieved, but got unexpected exit code 1'
            else:
                reason = str(e)
                result['failed'] = True
                result['msg'] = reason
        if reason:
            return result
        result['ping'] = 'pong'
        if warning:
            if 'warning' in result:
                result['warning'].append(warning)
            else:
                result['warning'] = [warning]
        return result