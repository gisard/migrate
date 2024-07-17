# migrate

migrate is a go data migration tool that can execute sql and go methods sequentially and return errors.
You need to specify the db connection and the schema table schemaTable, which is used to store the executed index and record error information.

# Directions
1. Run List
   - Migrations dir should have a schema file, for example migrate.txt, which is used to store execution list.
   - Execution list should contain all execution item which distinguish by file suffix, empty lines are not allowed.
   - .sql means it is a sql statement.
   - Empty suffix means it is a go method.
2. SQL Dir
   - Specify the sql file path freely, for example ./migrations
3. Go Method
   - Migrate client can apply structs or points, it will search go method from all applied structs or points.
   - Migrate exec go method by name and fill context by reflect.
   - Method format should be func(ctx context.Context) error.
4. Expand
   - You can expand other handlers by implement Handler interface.
   - Different handlers should be distinguished by suffix.
   - Add code when construct handlers of all type.