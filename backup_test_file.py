

import os
import subprocess
import sys


def get_paths(root):
    paths = []
    for path in os.listdir(root):
        if path.startswith('.'):
            continue

        path = root + '/' + path

        if os.path.isdir(path):
            paths = paths + get_paths(path)
        else:
            if path.endswith(endwith):
                paths.append(path)

    return paths


if __name__ == '__main__':

    if sys.argv[1] == 'backup':
        endwith = '_test.go'
        paths = get_paths(os.getcwd())
        for path in paths:
            print(path)
            # os.chdir(os.path.dirname(path))
            # filename = os.path.basename(path)
            subprocess.getstatusoutput('git mv {} {}'.format(path, path + '.back'))
            # os.rename(path, path + '.back')
    elif sys.argv[1] == 'restore':
        endwith = '_test.go.back'
        paths = get_paths(os.getcwd())
        for path in paths:
            print(path)
            # os.chdir(os.path.dirname(path))
            # filename = os.path.basename(path)
            subprocess.getstatusoutput('git mv {} {}'.format(path, path[:-5]))
            # os.rename(path, path[:-5])
