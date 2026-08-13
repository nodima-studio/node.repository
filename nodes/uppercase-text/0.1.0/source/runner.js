function process(row, config) {
  const value = row[config.column];
  if (value !== null && value !== undefined && typeof value !== "string") {
    throw new Error(`column ${config.column} is not a string`);
  }
  if (typeof value === "string") row[config.column] = value.toUpperCase();
  return row;
}
