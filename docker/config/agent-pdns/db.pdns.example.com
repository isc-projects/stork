; base zone file for pdns.example.com
$TTL 2d
$ORIGIN pdns.example.com.

@       IN      SOA     ns1.pdns.example.com. admin.pdns.example.com. (
                        2024031501      ; Serial
                        12h             ; Refresh
                        15m             ; Retry
                        3w              ; Expire
                        2h )            ; Minimum TTL

        IN      NS      ns1.pdns.example.com.
        IN      NS      ns2.pdns.example.com.
        IN      MX  10  mail.pdns.example.com.

; Static records
ns1     IN      A       10.0.0.2
ns2     IN      A       10.0.0.3
mail    IN      A       10.0.0.4
www     IN      A       10.0.0.5

; Generate records for web servers (web-1 through web-50)
$GENERATE 1-50 web-$ A 10.0.1.$

; Generate records for app servers (app-1 through app-50)
$GENERATE 1-50 app-$ A 10.0.2.$

; Generate records for db servers (db-1 through db-20)
$GENERATE 1-20 db-$ A 10.0.3.$

; The second name server is external to this zone (domain)
           IN      NS      ns2.pdns.example.net.
; Mail server RRs for the zone (domain)
        3w IN      MX  20  mail.pdns.example.net.
; Domain hosts include NS and MX records defined above
; plus any others required.
; For instance a user query for the A RR of joe.example.com will
; return the IPv4 address 192.168.254.6 from this zone file
ns1        IN      A       192.168.254.2
mail       IN      A       192.168.254.4
joe        IN      A       192.168.254.6
www        IN      A       192.168.254.7
; aliases ftp (ftp server) to an external domain
ftp        IN      CNAME   ftp.pdns.example.net.