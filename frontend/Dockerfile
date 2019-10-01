FROM nginx

COPY dist/stork /usr/share/nginx/html
COPY nginx.conf /tmp/nginx.conf.tpl

EXPOSE 80

ENV API_HOST localhost
ENV API_PORT 5000

CMD /bin/bash -c "DOLLAR=\$ envsubst < /tmp/nginx.conf.tpl > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"
