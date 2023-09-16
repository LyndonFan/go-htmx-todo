CREATE TABLE IF NOT EXISTS todos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT,
    created_date DATETIME,
    deadline_date DATETIME,
    status TEXT
);