CREATE TABLE song (
    id SERIAL PRIMARY KEY,
    song VARCHAR(255) NOT NULL,
    "group" VARCHAR(255) NOT NULL
);

CREATE TABLE songdetail (
    id SERIAL PRIMARY KEY,
    song_id INT NOT NULL,
    release_date DATE,
    text TEXT,
    link TEXT,
    FOREIGN KEY (song_id) REFERENCES song(id) ON DELETE CASCADE
);

CREATE INDEX idx_song_title ON song(song);
