package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	// Run the migrations (create tables)
	err = createTables(db)
	if err != nil {
		log.Fatal(err)
	}

	// Optionally, add mock data if the database is empty
	addMockData(db)

	return db
}

func createTables(db *sql.DB) error {
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        password_hash TEXT NOT NULL,
        email TEXT UNIQUE,
        role TEXT DEFAULT 'member',
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		bio TEXT,
		profile_image TEXT 
    );`

	createCategoriesTable := `
	CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        description TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	createPostsTable := `
    CREATE TABLE IF NOT EXISTS posts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        title TEXT NOT NULL,
        body TEXT NOT NULL,
        category_id INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id),
        FOREIGN KEY (category_id) REFERENCES categories(id)
    );`

	createCommentsTable := `
    CREATE TABLE IF NOT EXISTS comments (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        post_id INTEGER,
        user_id INTEGER,
        body TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (post_id) REFERENCES posts(id),
        FOREIGN KEY (user_id) REFERENCES users(id)
    );`

	createLikesTable := `
    CREATE TABLE IF NOT EXISTS likes_dislikes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        target_id INTEGER,
        target_type TEXT,  -- 'post' or 'comment'
        is_like BOOLEAN,   -- TRUE for like, FALSE for dislike
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id)
    );`

	createSessionsTable := `
    CREATE TABLE IF NOT EXISTS sessions (
		user_id INTEGER,
		email TEXT UNIQUE,
		session_token TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		
    );`

	_, err := db.Exec(createUsersTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createSessionsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createCategoriesTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createPostsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createCommentsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createLikesTable)
	if err != nil {
		return err
	}

	return nil
}

func addMockData(db *sql.DB) {
	// Check if any users exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		log.Println("Error checking user count:", err)
		return
	}

	if count == 0 {
		log.Println("No users found, inserting mock data...")

		// Insert mock users
		_, err = db.Exec(`INSERT INTO users (username, password_hash, email) VALUES
            ('alice', 'fakehashedpassword1', 'a@q'),
            ('bob', 'fakehashedpassword2', 'a@h'),
            ('carol', 'fakehashedpassword3', 'a@j');`)
		if err != nil {
			log.Println("Error inserting mock users:", err)
			return
		}

		// Inserting mock categories
		_, err = db.Exec(`INSERT INTO categories (id, name, description) VALUES
		(1, 'General Discussion', 'All about daily conversations and socializing.'),
		(2, 'Books & Reviews', 'Book discussions, reviews, and recommendations.'),
		(3, 'Literary Analysis', 'Deep dives into themes, symbols, and meanings in literature.'),
		(4, 'Writing & Creativity', 'Tips, exercises, and discussions for writers and creative minds.'),
		(5, 'Author Spotlight', 'Exploring the lives and works of notable authors.');`)
		if err != nil {
			log.Println("Error inserting mock categories:", err)
			return
		}

		// Inserting mock posts
		_, err = db.Exec(`INSERT INTO posts (id, user_id, title, body, category_id) VALUES
		(1, 1, 'How Fiction Mirrors Reality', 'Fiction often holds a mirror to our society, reflecting real-world issues like social justice, relationships, and the human condition. What are some books that you think do this effectively?', 1),
		(2, 2, 'The Role of Literature in Society', 'Literature has the power to shape ideas and influence culture. How do you think books have influenced major social movements?', 1),
		(3, 3, 'Modern Classics: Are They Worth the Hype?', 'There’s always a lot of discussion around "modern classics." Do you think recent novels like "The Road" by Cormac McCarthy or "Beloved" by Toni Morrison deserve their critical acclaim?', 1),

		(4, 1, 'Top Books of 2024', 'With so many amazing books set to release in 2024, it’s hard to know where to start. What are your most anticipated reads for this year?', 2),
		(5, 2, 'Underrated Authors You Should Know', 'Let’s talk about some authors who don’t always get the spotlight but have written incredible works. Who are some of your favorites?', 2),
		(6, 3, 'Book Review: The Midnight Library', 'Just finished "The Midnight Library" by Matt Haig. It’s a fascinating exploration of choices, regrets, and possibilities. Here’s my review and some thoughts on the themes of the book.', 2),

		(7, 1, 'Themes in Dystopian Novels', 'Dystopian literature often delves into themes like government control, individual freedom, and social oppression. Which dystopian books have left a lasting impact on you?', 3),
		(8, 2, 'Symbolism in "To Kill a Mockingbird"', 'Harper Lee’s "To Kill a Mockingbird" uses a lot of symbolism, from the mockingbird itself to the character of Atticus Finch. Let’s discuss some key symbols and what they mean to us.', 3),
		(9, 3, 'Exploring Existentialism in Modern Fiction', 'Existential themes have made a strong comeback in modern literature. Books like "Norwegian Wood" and "Fight Club" tackle the search for meaning. How do you interpret these themes?', 3),

		(10, 1, 'Writing Tips for Beginners', 'If you’re new to writing, here are some tips that can help you get started, from developing a writing routine to focusing on character development.', 4),
		(11, 2, 'How to Overcome Writer''s Block', 'Writer’s block can be frustrating, but there are ways to break through. Let’s share some techniques that have helped us stay creative and productive.', 4),
		(12, 3, 'Crafting Compelling Characters', 'A strong character can carry a story. Here are some tips for creating memorable and multi-dimensional characters that readers will love.', 4),

		(13, 1, 'The Legacy of Shakespeare', 'Shakespeare’s works have influenced generations of writers and artists. Which of his plays do you think holds the most relevance today, and why?', 5),
		(14, 2, 'Agatha Christie: The Queen of Mystery', 'Agatha Christie’s mysteries remain popular even decades after their release. What makes her stories so timeless, and which is your favorite?', 5),
		(15, 3, 'The Influence of J.K. Rowling on Modern Fantasy', 'Rowling’s Harry Potter series defined a generation. How do you think her work has impacted the fantasy genre as a whole?', 5);`)
		if err != nil {
			log.Println("Error inserting mock posts:", err)
			return
		}

		// Insert mock comments
		_, err = db.Exec(`INSERT INTO comments (post_id, user_id, body) VALUES
            (1, 2, 'Nice post, Alice!'),
            (2, 1, 'Thanks, Bob! Great thoughts!'),
            (3, 2, 'I completely agree with Carol''s points.');`)
		if err != nil {
			log.Println("Error inserting mock comments:", err)
			return
		}

		log.Println("Mock data inserted successfully.")
	}
}
