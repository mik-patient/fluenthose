FROM fluent/fluent-bit:1.8.3

ADD fluent-bit.conf /fluent-bit/etc/fluent-bit.conf
ADD parsers.conf /fluent-bit/etc/parsers.conf
ADD cloudfront.lua /fluent-bit/etc/scripts.lua
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 CMD curl -f http://localhost:2020/ || exit 1