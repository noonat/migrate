package migrate_test

import (
	"context"
	"database/sql"
	"log"

	"github.com/noonat/migrate"
)

// This example shows how you could define a set of simple SQL migrations
// using ExecQueries and then apply them to the database using Up. Note that
// you need to create an adapter for your database, so that migrate knows how
// to do things like specify placeholders. If your migration requires more
// advanced logic, you can also specify a custom MigrationFunc instead of a
// SQL string.
func ExampleUp() {
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
}
