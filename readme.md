# Judge-Dist

Judge-Dist should serve the purpose of sending judge tasks across actual judge machines. However,
currently I don't have enough time so Judge-dist will also judge the cases.

## Problem Format

```json
{
    "_id": "xxxxxxxxxxxxxx", // automatically generated
    "title": "NAOSI welcomes you!", // actual title of problem
    "timeLimit": "1800",  // second?
    "score": "20", 
    "tags": [
        "初尝算法"
    ], 
    "type": "algorithm",
    "data": {
        "content": {
            "body" : "输出 \"NAOSI welcomes you!\"（包括双引号）。",  // markdown should be supported
            "inputformat" : "⽆",
            "outputformat" : "⼀⾏。为题⽬所要求的字符串。",
            "sampleinput" : "",
            "sampleoutput" : "\"NAOSI welcomes you!\""
        }, 
        "restrictions": {
            "time" : 1000, // ms
            "mem" : 256000, // kb
        }, 
        "testcases" : [
            {"", "A/1.out"}
        ]
    }
}
```

## Endpoints

### POST /api/submit

Request content:

```json
{
    "code" : "xxxxxxxxxxxxxxxxxx", // base64 encoded string
    "_id" : "xxxxxxxxxxxxxx" // id declared previously in Problem Format
}
```

Response content:

```json
{
    "token" : "xxxxxxxxxxxxxxxxx" // uuid string
}
```

### GET /api/query?token=xxxxxxxxxxxxxxxxxxx

Response content:

```json
{
    "token" : "xxxxxxxxxxxxxxxxx", // uuid string,
    "testcases" : [
        {"num": 1, "status" : 0} // status: 0 - pending, 1 - accepted, 2 - compile error, 3 - compile timed out, 4 - runtime error, 5 - time limit exceeded, 6 - memory limit exceeded, 7 - output limit exceeded, 8 - wrong answer, 9 - other errors
    ]
}
```

### Configuration

```json
[
    {
        "_id" : "xxxxxxxxxxxxxx",
        "testcases" : [
            {"num" : 1, "input" : null, "output" : "A/1.out"}
        ]
    }
]
```