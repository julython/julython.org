nginx:
  pkg:
    - installed
  file.managed:
    - source: salt://nginx/conf/nginx.conf
    - name: /etc/nginx/nginx.conf
    - template: jinja
    - context:
      secrets: {{ pillar.secrets }}
    - require:
      - pkg: nginx
  service.running:
    - enable: True
    - watch:
      - file: /etc/nginx/nginx.conf
