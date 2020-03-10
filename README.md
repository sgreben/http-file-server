# http-file-server

`http-file-server` is a dependency-free HTTP file server. Beyond directory listings and file downloads, it lets you download a whole directory as as `.zip` or `.tar.gz` (generated on-the-fly).

![screenshot](doc/screenshot.png)

## Contents

- [Contents](#contents)
- [Examples](#examples)
  - [Serving a path at `/`](#serving-a-path-at-)
  - [Serving $PWD at `/`](#serving-pwd-at-)
  - [Serving multiple paths, setting the HTTP port via CLI arguments](#serving-multiple-paths-setting-the-http-port-via-cli-arguments)
  - [Setting the HTTP port via environment variables](#setting-the-http-port-via-environment-variables)
    - [Uploading files using cURL](#uploading-files-using-curl)
- [Get it](#get-it)
  - [Using `go get`](#using-go-get)
  - [Pre-built binary](#pre-built-binary)
- [Use it](#use-it)

## Examples

### Serving a path at `/`

```sh
$ http-file-server /tmp
2018/11/13 23:00:03 serving local path "/tmp" on "/tmp/"
2018/11/13 23:00:03 redirecting to "/tmp/" from "/"
2018/11/13 23:00:03 http-file-server listening on ":8080"
```

### Serving $PWD at `/`

```sh
$ cd /tmp
$ http-file-server
2018/12/13 03:18:00 serving local path "/tmp" on "/tmp/"
2018/12/13 03:18:00 redirecting to "/tmp/" from "/"
2018/12/13 03:18:00 http-file-server listening on ":8080"
```

### Serving multiple paths, setting the HTTP port via CLI arguments

```sh
$ http-file-server -p 1234 /1=/tmp /2=/var/tmp
2018/11/13 23:01:44 serving local path "/tmp" on "/1/"
2018/11/13 23:01:44 serving local path "/var/tmp" on "/2/"
2018/11/13 23:01:44 redirecting to "/1/" from "/"
2018/11/13 23:01:44 http-file-server listening on ":1234"
```

### Setting the HTTP port via environment variables

```sh
$ export PORT=9999
$ http-file-server /abc/def/ghi=/tmp
2018/11/13 23:05:52 serving local path "/tmp" on "/abc/def/ghi/"
2018/11/13 23:05:52 redirecting to "/abc/def/ghi/" from "/"
2018/11/13 23:05:52 http-file-server listening on ":9999"
```

#### Uploading files using cURL

```sh
$ ./http-file-server -uploads /=/path/to/serve
2020/03/10 22:00:54 serving local path "/path/to/serve" on "/"
2020/03/10 22:00:54 http-file-server listening on ":8080"
```

```sh
curl -LF "file=@example.txt" localhost:8080/path/to/upload/to
```

## Get it

### Using `go get`

```sh
go get -u github.com/sgreben/http-file-server
```

### Pre-built binary

Or [download a binary](https://github.com/sgreben/http-file-server/releases/latest) from the releases page, or from the shell:

```sh
# Linux
curl -L https://github.com/sgreben/http-file-server/releases/download/1.5.1/http-file-server_1.5.1_linux_x86_64.tar.gz | tar xz

# OS X
curl -L https://github.com/sgreben/http-file-server/releases/download/1.5.1/http-file-server_1.5.1_osx_x86_64.tar.gz | tar xz

# Windows
curl -LO https://github.com/sgreben/http-file-server/releases/download/1.5.1/http-file-server_1.5.1_windows_x86_64.zip
unzip http-file-server_1.5.1_windows_x86_64.zip
```

## Use it

```text
http-file-server [OPTIONS] [[ROUTE=]PATH] [[ROUTE=]PATH...]
```

```text
Usage of http-file-server:
  -a string
    	(alias for -addr) (default ":8080")
  -addr string
    	address to listen on (environment variable "ADDR") (default ":8080")
  -p int
    	(alias for -port)
  -port int
    	port to listen on (overrides -addr port) (environment variable "PORT")
  -q	(alias for -quiet)
  -quiet
    	disable all log output (environment variable "QUIET")
  -r value
    	(alias for -route)
  -route value
    	a route definition ROUTE=PATH (ROUTE defaults to basename of PATH if omitted)
  -u	(alias for -uploads)
  -uploads
    	allow uploads (environment variable "UPLOADS")
```
