
from setuptools import setup, find_packages

setup(
    name = 'july',
    version = '0.1.0',
    install_requires = [
        'django >= 1.5b1',
        'virtualenv >= 1.5.1',
    ],
    url = 'http://github.com/julython/julython.org',
    description = ("31 days and nights Python"),
    packages = find_packages(),
    test_suite = 'tests.run_tests',
)