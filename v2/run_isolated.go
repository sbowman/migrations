package migrations

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"
)

var ErrNoCommand = errors.New("no SQL command found")
var ErrNoState = errors.New("no SQL parser state")

// RequestChannel is the channel for submitting asynchronous migration requests.  Asynchronous
// migrations are run in the background, and must complete in order, but they do not wait for
// synchronous migrations to complete.
type RequestChannel chan AsyncRequest

// ResultChannel is the channel on which asynchronous migrations return their results (did they
// succeed or fail).  The MigrateAsync function returns this channel, so your application can
// listen for the background migrations to complete before reporting completion.
type ResultChannel chan AsyncResult

// AsyncRequest is submitted to the background asynchronous migration processor.
type AsyncRequest struct {
	Migration string    // Full path to the migration
	Direction Direction // The direction to run
	SQL       SQL       // The SQL to run (parsed from the migration)
	Target    int       // The desired revision number to migration to
}

// AsyncResult is returned by asynchronous SQL migration commands on the Results channel
type AsyncResult struct {
	Migration string // The migration filename
	Err       error  // Is nil if the migration succeeded
	Command   SQL    // The SQL command that failed if the migration failed; blank otherwise
}

// HandleAsync should be run in the background to listen for asynchronous migration requests.  These
// requests are run in order, within a transaction, but the main synchronous migrations will not
// wait for these to complete before continuing.
func HandleAsync(db *sql.DB, requests RequestChannel, results ResultChannel) {
	defer close(results)

	for req := range requests {
		Log.Infof("Running migration %s %s asynchronously", req.Migration, req.Direction)

		if cmd, err := RunIsolated(db, req); err != nil {
			results <- AsyncResult{req.Migration, err, cmd}
			continue
		}

		results <- AsyncResult{Migration: req.Migration}
	}
}

// RunIsolated breaks apart a SQL migration into separate commands and runs each in a single
// transaction.  Helps asynchronous migrations return additional details about failures.
func RunIsolated(db *sql.DB, req AsyncRequest) (SQL, error) {
	commands, err := ParseSQL(req.SQL)
	if err != nil {
		return "", err
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}

	for _, SQL := range commands {
		_, err = tx.Exec(string(SQL))
		if err != nil {
			_ = tx.Rollback()
			return SQL, err
		}
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return "", nil
}

// ParseSQL breaks the SQL document apart into individual commands, so we can submit them to the
// database one at a time.
func ParseSQL(doc SQL) ([]SQL, error) {
	var cmds []SQL

	parser := NewSQLParser(strings.TrimSpace(string(doc)))
	for parser.Next() {
		cmd, err := parser.Get()
		if err != nil {
			return nil, err
		}

		if cmd != "" {
			cmds = append(cmds, cmd)
		}
	}

	return cmds, nil
}

// SQLParser breaks apart a document of SQL commands into their individual commands.
type SQLParser struct {
	sql   string
	idx   int
	state []parserState
	cmd   []byte
	err   error
}

// NewSQLParser creats a new SQL parser.
func NewSQLParser(sql string) *SQLParser {
	return &SQLParser{
		sql: sql,
	}
}

// Next fetches the SQL command from from the document.
func (p *SQLParser) Next() bool {
	if p.idx == len(p.sql) {
		return false
	}

	p.cmd = p.cmd[:0]
	p.pushState(start)

	for {
		if p.idx == len(p.sql) || len(p.state) == 0 {
			return true
		}

		if err := p.fwd(); err != nil {
			p.err = err
			return true
		}
	}
}

// Get the SQL command parsed by Next().  Note that the semicolon will be stripped off.
func (p *SQLParser) Get() (SQL, error) {
	if p.cmd == nil {
		return "", ErrNoCommand
	} else if p.err != nil {
		return "", p.err
	}

	return SQL(bytes.TrimSpace(p.cmd)), nil
}

func (p *SQLParser) pushState(state parserState) {
	p.state = append(p.state, state)
}

func (p *SQLParser) popState() parserState {
	if len(p.state) == 0 {
		return nil
	}

	state := p.state[len(p.state)-1]
	p.state = p.state[:len(p.state)-1]

	return state
}

func (p *SQLParser) fwd() error {
	if len(p.state) == 0 {
		return ErrNoState // shouldn't happen, but just in case
	}

	state := p.state[len(p.state)-1]

	return state(p)
}

// Look ahead one character.
func (p *SQLParser) peek() byte {
	if p.idx == len(p.sql) {
		return 0
	}

	return p.sql[p.idx]
}

// Look ahead N characters.
func (p *SQLParser) peekN(n int) string {
	if p.idx+n >= len(p.sql) {
		n = len(p.sql) - p.idx
	}

	if n <= 0 {
		return ""
	}

	return p.sql[p.idx : p.idx+n]
}

// Get the next character and advance the index.
func (p *SQLParser) pop() byte {
	ch := p.sql[p.idx]
	p.cmd = append(p.cmd, ch)
	p.idx++
	return ch
}

// parserState is used to track the state of quoted strings in the SQL commands.
type parserState func(*SQLParser) error

func start(p *SQLParser) error {
	ch := p.peek()

	switch ch {
	case '\'':
		p.pop()
		p.pushState(single)
	case '"':
		p.pop()
		p.pushState(double)
	case '-':
		// Handle comments
		if p.peek() == '-' {
			for p.idx < len(p.sql) {
				p.idx = p.idx + 1
				if p.peek() == '\n' {
					break
				}
			}
			break
		}
		p.pop()
	case ';':
		p.idx = p.idx + 1
		p.popState()
	default:
		p.pop()
	}

	return nil
}

func single(p *SQLParser) error {
	ch := p.pop()

	switch ch {
	case '\\':
		if p.peek() == '\'' {
			p.pop()
		}
	case '\'':
		// Support ''' for escaping a single quote
		if p.peekN(2) == "''" {
			p.pop()
			p.pop()
			return nil
		}

		p.popState()
	default:
		// do nothing...
	}

	return nil
}

func double(p *SQLParser) error {
	ch := p.pop()

	switch ch {
	case '\\':
		if p.peek() == '"' {
			p.pop()
		}
	case '"':
		p.popState()
	default:
		// do nothing...
	}

	return nil
}
