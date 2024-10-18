# Judge-Dist

Judge-Dist should serve the purpose of sending judge tasks across actual judge machines. However,
currently I don't have enough time so Judge-dist will also judge the cases.

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