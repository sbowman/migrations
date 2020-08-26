package migrations_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/sbowman/migrations"
)

// Check that a single query parses correctly, with and without the semicolon.
func TestParseSingle(t *testing.T) {
	doc := migrations.SQL(`
create table sample(
	id serial primary key,
	name varchar(60) not null
)`)

	cmds, err := migrations.ParseSQL(doc)
	if err != nil {
		t.Errorf("Expected parse to succeed: %s", err)
	}

	if len(cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(cmds))
	}

	cmd := cmds[0]
	matching(t, cmd, "create table sample( id serial primary key, name varchar(60) not null )")

	// Try with a semicolon; should remove the semicolon
	doc = doc + ";"

	cmds, err = migrations.ParseSQL(doc)
	if err != nil {
		t.Errorf("Expected parse to succeed: %s", err)
	}

	if len(cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(cmds))
	}

	cmd = cmds[0]
	matching(t, cmd, "create table sample( id serial primary key, name varchar(60) not null )")
}

// Test that multiple queries come out ok.
func TestParseMulti(t *testing.T) {
	doc := migrations.SQL(`
create table sample(
	id serial primary key,
	name varchar(60) not null
);

create unique index idx_sample_name on sample (name);
`)

	cmds, err := migrations.ParseSQL(doc)
	if err != nil {
		t.Errorf("Expected parse to succeed: %s", err)
	}

	expected := []migrations.SQL{
		"create table sample( id serial primary key, name varchar(60) not null )",
		"create unique index idx_sample_name on sample (name)",
	}

	if len(cmds) != len(expected) {
		t.Errorf("Expected %d commands, got %d", len(expected), len(cmds))
	}

	for idx, cmd := range cmds {
		matching(t, cmd, expected[idx])
	}
}

// Test that quotes within quotes works.
func TestParseQuotes(t *testing.T) {
	doc := migrations.SQL(`
insert into sample ("dog's name") values ('Maya');
insert into sample ("dog--but no -- cats") values ('Maya');
insert into sample (name) values ('Maya "the dog" Dog');
insert into sample (location) values ('King\'s Ransom');
insert into sample (location) values ('King'''s Ransom');
insert into sample (phrase) values ('for whom; the bell tolls');
insert into sample (phrase) values ('for whom -- the bell tolls');
`)

	cmds, err := migrations.ParseSQL(doc)
	if err != nil {
		t.Errorf("Expected parse to succeed: %s", err)
	}

	expected := []migrations.SQL{
		`insert into sample ("dog's name") values ('Maya')`,
		`insert into sample ("dog--but no -- cats") values ('Maya')`,
		`insert into sample (name) values ('Maya "the dog" Dog')`,
		`insert into sample (location) values ('King\'s Ransom')`,
		`insert into sample (location) values ('King'''s Ransom')`,
		`insert into sample (phrase) values ('for whom; the bell tolls')`,
		`insert into sample (phrase) values ('for whom -- the bell tolls')`,
	}

	if len(cmds) != len(expected) {
		t.Errorf("Expected %d commands, got %d", len(expected), len(cmds))
	}

	for idx, cmd := range cmds {
		matching(t, cmd, expected[idx])
	}
}

func TestParseComments(t *testing.T) {
	doc := migrations.SQL(`
-- Creating a sample table
create table sample(
	id serial primary key,
	name varchar(60) not null -- name must be unique; don't overload this
);

-- Make sure the sample name is unique!
create unique index idx_sample_name on sample (name);

insert into sample (name) values ('--');

insert into sample ("weird -- column") values('hello');
`)

	cmds, err := migrations.ParseSQL(doc)
	if err != nil {
		t.Errorf("Expected parse to succeed: %s", err)
	}

	expected := []migrations.SQL{
		"create table sample( id serial primary key, name varchar(60) not null )",
		"create unique index idx_sample_name on sample (name)",
		"insert into sample (name) values ('--')",
		"insert into sample (\"weird -- column\") values('hello')",
	}

	if len(cmds) != len(expected) {
		t.Errorf("Expected %d commands, got %d", len(expected), len(cmds))
	}

	for idx, cmd := range cmds {
		matching(t, cmd, expected[idx])
	}
}

var spacing = regexp.MustCompile(`\s+`)
var feeds = regexp.MustCompile(`(?s)[\n\r]+`)

func matching(t *testing.T, cmd, expected migrations.SQL) bool {
	trimmed := spacing.ReplaceAllString(string(cmd), " ")
	single := feeds.ReplaceAllString(trimmed, "")

	matched := strings.EqualFold(single, string(expected))
	if !matched {
		t.Errorf(`Expected "%s", but got "%s"`, expected, single)
	}

	return matched
}
