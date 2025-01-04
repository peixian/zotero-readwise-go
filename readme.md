# Zotero to Readwise Converter
I got annoyed that all the other ones seemed to depend on python for some reason, and kept breaking as the environments moved on. This is a minimal one that only grabs what it needs. No deps.

## Running
```sh
peixian@toolbox ~/c/zotero-readwise (main)> go build .
peixian@toolbox ~/c/zotero-readwise (main)> ./zotero-readwise -h
Usage of ./zotero-readwise:
  -input string
        Input CSV file to send to Readwise
  -readwise-key string
        Readwise API key
  -zotero-key string
        Zotero API key
```
