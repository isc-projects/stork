; base zone file for bind9.example.com
$TTL 2d
$ORIGIN bind9.example.com.

@       IN      SOA     ns1.bind9.example.com. admin.bind9.example.com. (
                        2024031501      ; Serial
                        12h             ; Refresh
                        15m             ; Retry
                        3w              ; Expire
                        2h )            ; Minimum TTL

        IN      NS      ns1.bind9.example.com.
        IN      NS      ns2.bind9.example.com.
        IN      MX  10  mail.bind9.example.com.

; Static records
ns1     IN      A       11.0.0.2
ns2     IN      A       11.0.0.3
mail    IN      A       11.0.0.4
www     IN      A       11.0.0.5
