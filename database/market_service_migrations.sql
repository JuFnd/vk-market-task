CREATE TABLE IF NOT EXISTS adverts (
                                       id SERIAL PRIMARY KEY,
                                       title TEXT NOT NULL,
                                       description TEXT NOT NULL,
                                       price INT NOT NULL,
                                       image_path TEXT NOT NULL,
                                       profile_id INT NOT NULL,
                                       created_date DATE DEFAULT CURRENT_DATE
);
