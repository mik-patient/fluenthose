--[[ 
    parse and return the cloudfront headers
 ]]
-- capture the name searched for by require
local NAME=...

-- table for our functions
local M = { }


-- main function
function parseCloudfrontHeaders(tag, timestamp, record)
    local rec_type = record["type"]
    if (rec_type == nil) then
        return 0, 0, 0
    end
    if (rec_type == "cloudfront") then
        local new_record = record
        local csHeaderNames = Unescape(record["csHeaderNames"])
        local csHeaders = Unescape(record["csHeaders"])
        local csCookie = Unescape(record["csCookie"])
        local csUserAgent = Unescape(record["csUserAgent"])

        new_record["csUserAgentParsed"] = csUserAgent
        new_record["csCookieParsed"] = CookieParser(csCookie)
        new_record["csHeadersParsed"] = HeaderParser(csHeaders)
        new_record["csHeaderNamesParsed"] = csHeaderNames
        return 2, timestamp, new_record
    else
        return 0, 0, 0
    end
end

Hex_to_char = function(x)
    return string.char(tonumber(x, 16))
end

Unescape = function(urlEncoded)
    if (urlEncoded == nil) then
        return ""
    end
    local urlDecoded = urlEncoded:gsub("%%(%x%x)", Hex_to_char)
    return urlDecoded
    -- return urlDecoded:gsub("\n", " ")
end

HeaderParser = function (x)
    local c = SplitHeaders(x)
    return c
    
end

SplitHeaders = function (x)
    local result = {}
    for line, v in x:gmatch("[^\r\n]+") do
        local key, value = line:match("^([^:]+):%s*(.+)$")
        if key then
            result[key:lower()] = value
        end
    end
    return result
    
end

CookieParser = function (x)
    local c = SplitCookies(x)
    return c
end

SplitCookies = function (x)
    local result = {}
    for k, v in x:gmatch("([^;%s]+)=([^;%s]+)") do
        if (k ~=  nil) then
            result[k:lower()] = Unescape(v)
        end
    end
    return result
end

function Dump(o)
    if type(o) == 'table' then
       local s = '{ '
       for k,v in pairs(o) do
          if type(k) ~= 'number' then k = '"'..k..'"' end
          s = s .. '['..k..'] = ' .. Dump(v) .. ',\n '
       end
       return s .. '} '
    else
       return tostring(o)
    end
 end

-- end function

-- export functions
M.parseCloudfrontHeaders = parseCloudfrontHeaders
M.SplitHeaders = SplitHeaders
return M