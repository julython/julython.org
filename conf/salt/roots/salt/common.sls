#
# There is a bug in salt which requires that we install
# this package first otherwise the special apt repo 
# addition will fail.

python-software-properties:
  pkg.installed