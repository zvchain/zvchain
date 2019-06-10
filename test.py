

import os
import re
import subprocess

ignore_paths = ['.*/deploy', '.*/lib', '.*/tvm/py', '.*/tvm/ctvm', '.*/cmd/gtas/fronted*']


def get_dirs(root):
    paths = []
    for path in os.listdir(root):
        if path.startswith('.'):
            continue

        path = root + '/' + path

        if os.path.isdir(path):
            ignore = False
            for ignore_path in ignore_paths:
                if re.match(re.compile(ignore_path), path) is not None:
                    # print(path)
                    ignore = True
                    break
                # if path.endswith(ignore_path):
                #     ignore = True
                #     break
            if not ignore:
                paths.append(path)
            paths = paths + get_dirs(path)
    return paths


if __name__ == '__main__':

    dirs = get_dirs(os.getcwd())

    for dir_path in dirs:
        print("\033[1;31;m", dir_path, '\033[0m')
        os.chdir(dir_path)
        # os.system('go test')

        output = subprocess.getstatusoutput('go test')
        for line in output:
            print(line)
