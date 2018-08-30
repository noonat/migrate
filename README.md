# migrate

[![godoc](https://godoc.org/github.com/noonat/migrate?status.svg)][godoc]
[![travis](https://travis-ci.org/noonat/migrate.svg)][travis]
[![report](https://goreportcard.com/badge/github.com/noonat/migrate)][report]

Package migrate provides helpers for running SQL database migrations. It's
designed for migrations that are specified in code and distributed as part
of the application binary, and applied as part of the application startup
(rather than via external files and an external tool).

## Usage

You can read the [package documentation][godoc] for more information, but here
is a simple example:

```go
adapter := migrate.NewPostgreSQLAdapter(log.Printf)
db, err := sql.Open("migrate_test", "")
if err != nil {
    log.Panic(err)
}

migrations := []migrate.Migration{
    {
        Comment: "Add user and app tables",
        Up: migrate.ExecQueries([]string{
            `CREATE TABLE users (
                id SERIAL PRIMARY KEY,
                username VARCHAR(50) NOT NULL)`,
            `CREATE TABLE apps (
                id SERIAL PRIMARY KEY,
                title VARCHAR(100) NOT NULL)`,
        }),
        Down: migrate.ExecQueries([]string{
            `DROP TABLE apps`,
            `DROP TABLE users`,
        }),
    },
    {
        Comment: "Add user app join table",
        Up: migrate.ExecQueries([]string{
            `CREATE TABLE user_apps (
                user_id INT NOT NULL REFERENCES users(id),
                app_id INT NOT NULL REFERENCES apps(id),
                UNIQUE (user_id, app_id))`,
        }),
        Down: migrate.ExecQueries([]string{
            `DROP TABLE user_apps`,
        }),
    },
}

err = migrate.Up(context.Background(), db, adapter, migrations)
if err != nil {
    log.Panicf("error running migrations: %s", err)
}
```

## License

MIT

[travis]: https://travis-ci.org/noonat/migrate
[report]: https://goreportcard.com/report/github.com/noonat/migrate
[godoc]: https://godoc.org/github.com/noonat/migrate
