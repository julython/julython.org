nginx:
  pkg:
    - installed
  service.running:
    - enable: True
    - watch:
      - file: /etc/nginx/nginx.conf

/etc/nginx/nginx.conf:
  file.managed:
    - source: salt://nginx/conf/nginx.conf
    - template: jinja
    - context:
      secrets: {{ pillar.secrets }}
    - require:
      - pkg: nginx

/etc/nginx/sites-enabled:
  file.recurse:
    - source: salt://nginx/conf/sites-enabled
    - template: jinja
    - makedirs: True
    - context:
      secrets: {{ pillar.secrets }}
    - require:
      - pkg: nginx
