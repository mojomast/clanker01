package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/swarm-ai/swarm/internal/skills/loader"
)

type Skill struct {
	manifest *loader.SkillManifest
	db       *sql.DB
}

func NewSkill() *Skill {
	manifest := &loader.SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: loader.SkillMetadata{
			Name:        "database",
			Version:     "1.0.0",
			DisplayName: "Database Operations",
			Description: "Query databases, execute statements, and manage schemas",
			Author:      "SWARM Team",
			License:     "Apache-2.0",
			Tags:        []string{"database", "sql", "storage"},
		},
		Spec: loader.SkillSpec{
			Runtime:    "native",
			Entrypoint: "builtin.database",
			Tools: []loader.ToolDef{
				{
					Name:        "query",
					Description: "Execute a SELECT query and return results",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"driver", "dsn", "sql"},
						"properties": map[string]interface{}{
							"driver": map[string]string{
								"type":        "string",
								"description": "Database driver (mysql, postgres, sqlite, etc.)",
							},
							"dsn": map[string]string{
								"type":        "string",
								"description": "Data source name/connection string",
							},
							"sql": map[string]string{
								"type":        "string",
								"description": "SQL query to execute",
							},
							"args": map[string]interface{}{
								"type":        "array",
								"items":       map[string]string{"type": "string"},
								"description": "Query arguments for prepared statements",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"columns":   map[string]string{"type": "array"},
							"rows":      map[string]string{"type": "array"},
							"row_count": map[string]string{"type": "integer"},
						},
					},
				},
				{
					Name:        "execute",
					Description: "Execute a non-query SQL statement (INSERT, UPDATE, DELETE, etc.)",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"driver", "dsn", "sql"},
						"properties": map[string]interface{}{
							"driver": map[string]string{
								"type":        "string",
								"description": "Database driver (mysql, postgres, sqlite, etc.)",
							},
							"dsn": map[string]string{
								"type":        "string",
								"description": "Data source name/connection string",
							},
							"sql": map[string]string{
								"type":        "string",
								"description": "SQL statement to execute",
							},
							"args": map[string]interface{}{
								"type":        "array",
								"items":       map[string]string{"type": "string"},
								"description": "Statement arguments for prepared statements",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"rows_affected": map[string]string{"type": "integer"},
							"success":       map[string]string{"type": "boolean"},
						},
					},
				},
				{
					Name:        "schema",
					Description: "Get database schema information",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"driver", "dsn"},
						"properties": map[string]interface{}{
							"driver": map[string]string{
								"type":        "string",
								"description": "Database driver (mysql, postgres, sqlite, etc.)",
							},
							"dsn": map[string]string{
								"type":        "string",
								"description": "Data source name/connection string",
							},
							"table": map[string]string{
								"type":        "string",
								"description": "Specific table to get schema for (optional)",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tables":  map[string]string{"type": "array"},
							"columns": map[string]string{"type": "array"},
						},
					},
				},
				{
					Name:        "transaction",
					Description: "Execute multiple statements in a transaction",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"driver", "dsn", "statements"},
						"properties": map[string]interface{}{
							"driver": map[string]string{
								"type":        "string",
								"description": "Database driver (mysql, postgres, sqlite, etc.)",
							},
							"dsn": map[string]string{
								"type":        "string",
								"description": "Data source name/connection string",
							},
							"statements": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"sql": map[string]string{"type": "string"},
										"args": map[string]interface{}{
											"type":  "array",
											"items": map[string]string{"type": "string"},
										},
									},
								},
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]string{"type": "boolean"},
							"results": map[string]string{"type": "array"},
						},
					},
				},
			},
			Config: loader.SkillConfig{
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"default_driver": map[string]string{"type": "string"},
						"default_dsn":    map[string]string{"type": "string"},
						"timeout": map[string]interface{}{
							"type":    "integer",
							"default": 30,
						},
					},
				},
			},
			Permissions: loader.SkillPermissions{
				Filesystem: loader.FilesystemPermissions{
					Read:   []string{},
					Write:  []string{},
					Delete: []string{},
				},
				Network: loader.NetworkPermissions{
					Allow:        true,
					AllowedHosts: []string{"*"},
				},
			},
		},
	}
	return &Skill{manifest: manifest}
}

func (s *Skill) Meta() *loader.SkillManifest {
	return s.manifest
}

func (s *Skill) Initialize(ctx context.Context, config *loader.Config) error {
	return nil
}

func (s *Skill) Shutdown(ctx context.Context) error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Skill) Tools() []loader.Tool {
	var tools []loader.Tool
	for _, t := range s.manifest.Spec.Tools {
		tools = append(tools, loader.Tool{
			Type: "function",
			Function: loader.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return tools
}

func (s *Skill) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*loader.Result, error) {
	switch toolName {
	case "query":
		return s.query(ctx, args)
	case "execute":
		return s.execute(ctx, args)
	case "schema":
		return s.schema(ctx, args)
	case "transaction":
		return s.transaction(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (s *Skill) getDB(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (s *Skill) query(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	driver, ok := args["driver"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "driver is required"}, nil
	}

	dsn, ok := args["dsn"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "dsn is required"}, nil
	}

	sqlStr, ok := args["sql"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "sql is required"}, nil
	}

	db, err := s.getDB(ctx, driver, dsn)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer db.Close()

	queryArgs, _ := args["args"].([]interface{})

	var rows *sql.Rows
	var queryErr error

	if len(queryArgs) > 0 {
		argsInterface := make([]interface{}, len(queryArgs))
		for i, arg := range queryArgs {
			argsInterface[i] = arg
		}
		rows, queryErr = db.QueryContext(ctx, sqlStr, argsInterface...)
	} else {
		rows, queryErr = db.QueryContext(ctx, sqlStr)
	}

	if queryErr != nil {
		return &loader.Result{Success: false, Error: queryErr.Error()}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	var rowData [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}

		rowData = append(rowData, values)
	}

	result := map[string]interface{}{
		"columns":   columns,
		"rows":      rowData,
		"row_count": len(rowData),
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) execute(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	driver, ok := args["driver"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "driver is required"}, nil
	}

	dsn, ok := args["dsn"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "dsn is required"}, nil
	}

	sqlStr, ok := args["sql"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "sql is required"}, nil
	}

	db, err := s.getDB(ctx, driver, dsn)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer db.Close()

	execArgs, _ := args["args"].([]interface{})

	var result sql.Result
	var execErr error

	if len(execArgs) > 0 {
		argsInterface := make([]interface{}, len(execArgs))
		for i, arg := range execArgs {
			argsInterface[i] = arg
		}
		result, execErr = db.ExecContext(ctx, sqlStr, argsInterface...)
	} else {
		result, execErr = db.ExecContext(ctx, sqlStr)
	}

	if execErr != nil {
		return &loader.Result{Success: false, Error: execErr.Error()}, nil
	}

	rowsAffected, _ := result.RowsAffected()

	response := map[string]interface{}{
		"rows_affected": rowsAffected,
		"success":       true,
	}

	return &loader.Result{Success: true, Data: response}, nil
}

func (s *Skill) schema(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	driver, ok := args["driver"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "driver is required"}, nil
	}

	dsn, ok := args["dsn"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "dsn is required"}, nil
	}

	table, _ := args["table"].(string)

	db, err := s.getDB(ctx, driver, dsn)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer db.Close()

	var tables []map[string]interface{}

	if table == "" {
		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				tables = append(tables, map[string]interface{}{
					"name": name,
				})
			}
		}
	} else {
		tables = append(tables, map[string]interface{}{
			"name": table,
		})
	}

	var columns []map[string]interface{}
	for _, t := range tables {
		tableName := t["name"].(string)

		rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}

		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err == nil {
				columns = append(columns, map[string]interface{}{
					"table":       tableName,
					"name":        name,
					"type":        dataType,
					"not_null":    notNull == 1,
					"default":     dfltValue,
					"primary_key": pk == 1,
				})
			}
		}
		rows.Close()
	}

	result := map[string]interface{}{
		"tables":  tables,
		"columns": columns,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) transaction(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	driver, ok := args["driver"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "driver is required"}, nil
	}

	dsn, ok := args["dsn"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "dsn is required"}, nil
	}

	statements, ok := args["statements"].([]interface{})
	if !ok {
		return &loader.Result{Success: false, Error: "statements is required"}, nil
	}

	db, err := s.getDB(ctx, driver, dsn)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	var results []map[string]interface{}
	success := true

	for _, stmt := range statements {
		stmtMap, ok := stmt.(map[string]interface{})
		if !ok {
			tx.Rollback()
			return &loader.Result{Success: false, Error: "invalid statement format"}, nil
		}

		sqlStr, ok := stmtMap["sql"].(string)
		if !ok {
			tx.Rollback()
			return &loader.Result{Success: false, Error: "statement missing sql field"}, nil
		}

		stmtArgs, _ := stmtMap["args"].([]interface{})

		var result sql.Result
		var execErr error

		if len(stmtArgs) > 0 {
			argsInterface := make([]interface{}, len(stmtArgs))
			for i, arg := range stmtArgs {
				argsInterface[i] = arg
			}
			result, execErr = tx.ExecContext(ctx, sqlStr, argsInterface...)
		} else {
			result, execErr = tx.ExecContext(ctx, sqlStr)
		}

		resultEntry := map[string]interface{}{
			"sql": sqlStr,
		}

		if execErr != nil {
			resultEntry["error"] = execErr.Error()
			resultEntry["success"] = false
			success = false
			tx.Rollback()
			break
		} else {
			rowsAffected, _ := result.RowsAffected()
			resultEntry["rows_affected"] = rowsAffected
			resultEntry["success"] = true
		}

		results = append(results, resultEntry)
	}

	if success {
		if err := tx.Commit(); err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}
	}

	response := map[string]interface{}{
		"success": success,
		"results": results,
	}

	return &loader.Result{Success: true, Data: response}, nil
}
