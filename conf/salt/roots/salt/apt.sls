base:
  pkgrepo.managed:
    - humanname: Example APT Repo
    - name: deb  http://mirror.ubuntu.com
    - file: /etc/apt/sources.list.d/mirror.list
    - require:
      - sls: common