--- tests (brew install lua && luarocks && luarocks install luaunit)
Lu = require('luaunit')
TestCloudfront = {}
    function TestCloudfront:setUp()
        self.tag = "test"
        self.timestamp = 12345
        self.record = {
            ["type"] = "cloudfront",
            ["csHost"] = "example.com",
            ["csUriStem"] = "/index.html",
            ["csUriQuery"] = "-",
            ["csMethod"] = "GET",
            ["csCookie"] = "_ga=GA1.1.245364568.15967356180;%20SOME-MARKET=en;%20_uetvid=c1b4d290eba8bff13d646a52cb;%20_gcl_au=1.1.157529467680.16306rr37;%20_ga_BBLDER84WH=GS1.1.1633963456167.3.1.1633963161.0;%20CONSENT=%257B%2522functional%2522%253A1%252C%2522marketing%2522%253A1%252C%2522performance%2522%253A1%252C%2522targeting%2522%253A1%257D;%20SOME-LOBBY=fobar",
            ["csHeaderNames"] = "if-none-match%0Acookie%0Aaccept-language%0Aaccept-encoding%0Areferer%0Asec-fetch-dest%0Asec-fetch-mode%0Asec-fetch-site%0Aaccept%0Asec-ch-ua-platform%0Auser-agent%0Asec-ch-ua-mobile%0Asec-ch-ua%0Aconnection%0Ahost%0ACloudFront-Viewer-Country%0ACloudFront-Viewer-Country-Region%0A",
            ["csHeaders"] = "if-none-match:W/%2226b-NM56YxcY1QHuHPTLEZz4uWLaS9s%22%0Acookie:xxx%0Aaccept-language:en-GB,en-US;q=0.9,en;q=0.8%0Aaccept-encoding:gzip,%20deflate,%20br%0Areferer:http://localhost:4200/en%0Asec-fetch-dest:empty%0Asec-fetch-mode:cors%0Asec-fetch-site:same-origin%0Aaccept:*/*%0Asec-ch-ua-platform:%22Windows%22%0Auser-agent:Mozilla/5.0%20(Windows%20NT%2010.0;%20Win64;%20x64)%20AppleWebKit/537.36%20(KHTML,%20like%20Gecko)%20Chrome/96.0.4664.45%20Safari/537.36%0Asec-ch-ua-mobile:?0%0Asec-ch-ua:%22%20Not%20A;Brand%22;v=%2299%22,%20%22Chromium%22;v=%2296%22,%20%22Google%20Chrome%22;v=%2296%22%0A",
            ["csUserAgent"] = "Mozilla/5.0%20(Windows%20NT%2010.0;%20Win64;%20x64)%20AppleWebKit/537.36%20(KHTML,%20like%20Gecko)%20Chrome/96.0.4664.45%20Safari/537.36"
        }
    end
    function TestCloudfront:testCloudfront()
        local cloudfront = require('cloudfront')
        local code, ts, rec = cloudfront.parseCloudfrontHeaders(self.tag, self.timestamp, self.record)
        Lu.assertEquals(code, 2)
        Lu.assertEquals(ts, 12345)
        Lu.assertEquals(rec.csHost, "example.com")
        Lu.assertEquals(rec.csUriStem, "/index.html")
        Lu.assertEquals(rec.csUriQuery, "-")
        Lu.assertEquals(rec.csMethod, "GET")
        Lu.assertEquals(rec.csCookieParsed["some-market"], "en")
        Lu.assertEquals(rec.csCookieParsed["some-lobby"], "fobar")
        Lu.assertEquals(rec.csHeadersParsed["if-none-match"], "W/\"26b-NM56YxcY1QHuHPTLEZz4uWLaS9s\"")
        Lu.assertEquals(rec.csHeadersParsed["cookie"], "xxx")
    end
    function TestCloudfront:testCloudfrontWithNilHeaders()
        local cloudfront = require('cloudfront')
        local record = {
            ["type"] = "cloudfront",
            ["csHost"] = "example.com",
            ["csUriStem"] = "/index.html",
            ["csUriQuery"] = "-",
            ["csMethod"] = "GET",
        }
        local result = cloudfront.parseCloudfrontHeaders(self.tag, self.timestamp, record)
    end

    function TestCloudfront:testCloudfrontWithWrongType()
        local cloudfront = require('cloudfront')
        local record = {
            ["type"] = "wrong",
            ["csHost"] = "example.com",
            ["csUriStem"] = "/index.html",
            ["csUriQuery"] = "-",
            ["csMethod"] = "GET",
        }
        local result = cloudfront.parseCloudfrontHeaders(self.tag, self.timestamp, record)
        Lu.assertEquals(result, 0)
    end 
os.exit( Lu.LuaUnit.run() )