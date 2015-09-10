# Ballad

HTTP edit line tool.

Ballad send HTTP request to url input by stdin.

And edit input line and output to stdout by HTTP response.

## Usage

```shell
$ cat data.txt
https://www.google.co.jp/
https://www.google.co.jp/a
https://www.google.co.jp/b

$ cat data.txt | ballad
200	https://www.google.co.jp/
301	https://www.google.co.jp/a
404	https://www.google.co.jp/b

$ cat data.txt | ballad | grep 200 | awk '{print $2}'
https://www.google.co.jp/

```

## Installation

```shell
$ git clone https://github.com/ksss/ballad-go
$ cd ballad-go
$ go build -o ballad
$ go install
```
