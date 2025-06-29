#!/bin/bash

# migrate.sh - Database migration helper script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
MIGRATIONS_DIR="./migrations"
COMMAND=""

# Help function
show_help() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  up              Apply all pending migrations"
    echo "  down            Rollback the last migration"
    echo "  down-all        Rollback all migrations"
    echo "  create NAME     Create a new migration with the given name"
    echo "  status          Show migration status"
    echo "  force VERSION   Force database schema to specific version"
    echo "  version         Show current database version"
    echo ""
    echo "Environment variables:"
    echo "  DB_DSN          Database connection string (required)"
    echo ""
    echo "Examples:"
    echo "  $0 up"
    echo "  $0 create add_user_roles"
    echo "  $0 down"
}

# Check if migrate tool is installed
check_migrate_installed() {
    if ! command -v migrate &> /dev/null; then
        echo -e "${RED}Error: 'migrate' tool is not installed${NC}"
        echo "Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        exit 1
    fi
}

# Check database connection
check_db_connection() {
    if [ -z "$DB_DSN" ]; then
        echo -e "${RED}Error: DB_DSN environment variable is not set${NC}"
        echo "Please set it to your PostgreSQL connection string"
        echo "Example: export DB_DSN='postgres://user:password@localhost:5432/dbname?sslmode=disable'"
        exit 1
    fi
}

# Run migration command
run_migrate() {
    local cmd=$1
    shift
    
    echo -e "${YELLOW}Running: migrate -path $MIGRATIONS_DIR -database \"$DB_DSN\" $cmd $@${NC}"
    migrate -path "$MIGRATIONS_DIR" -database "$DB_DSN" $cmd "$@"
}

# Main script
main() {
    check_migrate_installed
    
    # Parse command
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi
    
    COMMAND=$1
    shift
    
    case $COMMAND in
        up)
            check_db_connection
            echo -e "${GREEN}Applying all pending migrations...${NC}"
            run_migrate up
            echo -e "${GREEN}Migrations applied successfully${NC}"
            ;;
            
        down)
            check_db_connection
            echo -e "${YELLOW}Rolling back last migration...${NC}"
            run_migrate down 1
            echo -e "${GREEN}Rollback completed${NC}"
            ;;
            
        down-all)
            check_db_connection
            echo -e "${YELLOW}Rolling back all migrations...${NC}"
            read -p "Are you sure? This will delete all data! (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                run_migrate down -all
                echo -e "${GREEN}All migrations rolled back${NC}"
            else
                echo -e "${YELLOW}Cancelled${NC}"
            fi
            ;;
            
        create)
            if [ $# -eq 0 ]; then
                echo -e "${RED}Error: Migration name is required${NC}"
                echo "Usage: $0 create <migration_name>"
                exit 1
            fi
            
            NAME=$1
            echo -e "${GREEN}Creating migration: $NAME${NC}"
            migrate create -ext sql -dir "$MIGRATIONS_DIR" -seq "$NAME"
            echo -e "${GREEN}Migration files created${NC}"
            ;;
            
        status)
            check_db_connection
            echo -e "${GREEN}Migration status:${NC}"
            run_migrate version
            ;;
            
        force)
            check_db_connection
            if [ $# -eq 0 ]; then
                echo -e "${RED}Error: Version number is required${NC}"
                echo "Usage: $0 force <version>"
                exit 1
            fi
            
            VERSION=$1
            echo -e "${YELLOW}Forcing database to version: $VERSION${NC}"
            read -p "Are you sure? This can cause data inconsistency! (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                run_migrate force "$VERSION"
                echo -e "${GREEN}Database forced to version $VERSION${NC}"
            else
                echo -e "${YELLOW}Cancelled${NC}"
            fi
            ;;
            
        version)
            check_db_connection
            run_migrate version
            ;;
            
        *)
            echo -e "${RED}Error: Unknown command '$COMMAND'${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"