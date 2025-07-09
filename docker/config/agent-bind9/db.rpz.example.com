$TTL 300

@       IN SOA  localhost. need.to.know.only. (
                       201702121 ; Serial number
                       60        ; Refresh every minute
                       60        ; Retry every minute
                       432000    ; Expire in 5 days
                       60 )      ; negative caching ttl 1 minute
        IN NS   LOCALHOST.

; Block queries to example.com.
example.com             IN CNAME .
*.example.com           IN CNAME .
