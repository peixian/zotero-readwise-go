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

# License
Apache 2. Go nuts.

```
Copyright 2025 Peixian Wang

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
