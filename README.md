# ff7r-text-tool

Text modding tool for FF7R trilogy.

> [!warning]
> This repository is still a WIP project. There might be tons of issues.

## Features

- Export text data from `*_TxtRes.uasset` as csv or json
- Import text data into `*_TxtRes.uasset`
- Some utilities for [my dual-subtitle mods](https://www.nexusmods.com/finalfantasy7rebirth/mods/79)

## Changes from [my old tool](https://github.com/matyamod/FF7R_text_mod_tools)

- Executable is now smaller, faster, and safer than the python script.
- Exported data has more reasonable format.
- Missing entries now do not throw erros when importing.
- Supported CSV format.
- Supported FF7R2 assets (that extracted from Fmodel).

## How to use

Launch `GUI.exe` and specify paths for exporting.  
![screen_shot](https://github.com/user-attachments/assets/9b0eb221-9c19-43ad-9771-51a604bd53c4)

You can also activate other modes from the menu bar.  
![menu](https://github.com/user-attachments/assets/900591d3-4acd-4d19-b9e9-734eb6db4979)

## How to build

### Build CLI program

Install [go 1.23.5](https://go.dev/doc/install). (I'm not sure if other versions work or not)  
Then, run the following commands in the git repository

```
go get github.com/spf13/pflag
go build -ldflags="-s -w" -trimpath
```

### Get GUI wrapper

Download `Tuw-*-Windows10-x64.zip` from [here](https://github.com/matyalatte/tuw/releases).  
Then, copy `Tuw.exe` to the git repository and rename it to `GUI.exe`.


## CSV example

CSV makes it possible for you to edit text data with spreadsheet programs but it can't keep the original nested structure.

```
id,sub_id,text
language,,US
$foo_bar_0000,,Hello!<br>Hello!
$foo_bar_0000,ACTOR,"Your mom"
```

## JSON example

JSON can keep the original structure but it may hard to edit it manually.

```json
{
  "language": "US",
  "entries": [
    {
      "id": "$foo_bar_0000",
      "text": "Hello!\r\nHello!",
      "sub_entries": [
        {
          "id": "ACTOR",
          "text": "Your mom"
        }
      ]
    },
  ]
}
```
