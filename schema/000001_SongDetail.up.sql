CREATE TABLE Song (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    "group" VARCHAR(255) NOT NULL
);

CREATE TABLE SongDetail (
    id SERIAL PRIMARY KEY,
    song_id INT NOT NULL,
    release_date DATE,
    text TEXT,
    link VARCHAR(500),
    FOREIGN KEY (song_id) REFERENCES Song(id) ON DELETE CASCADE
);