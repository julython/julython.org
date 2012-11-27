#!/usr/bin/env python
import os, sys

from gae_django.fabric_commands import setup_paths

setup_paths()

if __name__ == "__main__":
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "july.settings")

    from django.core.management import execute_from_command_line
    
    # Don't allow runserver command
    if len(sys.argv) > 1:
        if sys.argv[1] == 'runserver':
            print("Use appengine dev_appserver.py or fabric to runserver!")
            sys.exit(1)
    
    execute_from_command_line(sys.argv)
