-- Config holds global config for the Markov chain data.
CREATE TABLE Config (
    option TEXT NOT NULL,
    value ANY
) STRICT;

INSERT INTO Config(option, value) VALUES
    ('schema-version', {{$.Version}}),
    ('order', {{$.N}});
