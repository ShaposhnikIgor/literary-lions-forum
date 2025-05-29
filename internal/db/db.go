package database

import (
	"database/sql" // Import the package for database operations.
	"log"          // Import the package for logging errors or messages.

	_ "github.com/mattn/go-sqlite3" // Import SQLite3 driver for database interaction (side-effect import).
)

// InitDB initializes the database connection and sets up the schema.
func InitDB(filepath string) *sql.DB {
	// Open a connection to the SQLite database using the provided file path.
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		// Log a fatal error and terminate the application if the connection fails.
		log.Fatal(err)
	}

	// Create necessary tables if they do not exist.
	err = createTables(db)
	if err != nil {
		// Log a fatal error and terminate the application if table creation fails.
		log.Fatal(err)
	}

	// Optionally populate the database with mock data if it is empty.
	addMockData(db)

	// Return the database connection object for use in the application.
	return db
}

// createTables defines and executes SQL queries to create required tables.
func createTables(db *sql.DB) error {
	// SQL query to create the `users` table if it does not already exist.
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the user.
        username TEXT NOT NULL UNIQUE,        -- Unique username for the user.
        password_hash TEXT NOT NULL,          -- Encrypted password for authentication.
        email TEXT UNIQUE,                    -- Optional unique email address for the user.
        role TEXT DEFAULT 'member',           -- Role of the user, e.g., admin or member.
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp of when the user was created.
        bio TEXT,                             -- Optional biography of the user.
        profile_image TEXT DEFAULT 'assets/static/images/placeholder.png' -- Default profile image.
    );`

	// SQL query to create the `categories` table if it does not already exist.
	createCategoriesTable := `
	CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the category.
        name TEXT NOT NULL,                   -- Name of the category.
        description TEXT,                     -- Optional description of the category.
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP -- Timestamp of when the category was created.
    );`

	// SQL query to create the `posts` table if it does not already exist.
	createPostsTable := `
    CREATE TABLE IF NOT EXISTS posts (
        id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the post.
        user_id INTEGER,                      -- ID of the user who created the post.
        title TEXT NOT NULL,                  -- Title of the post.
        body TEXT NOT NULL,                   -- Content of the post.
        category_id INTEGER,                  -- ID of the category the post belongs to.
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp of when the post was created.
        FOREIGN KEY (user_id) REFERENCES users(id),    -- Relationship to the "user" table.
        FOREIGN KEY (category_id) REFERENCES categories(id) -- Relationship to the "categories" table.
    );`

	// SQL query to create the `comments` table if it does not already exist.
	createCommentsTable := `
    CREATE TABLE IF NOT EXISTS comments (
        id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the comment.
        post_id INTEGER,                      -- ID of the post the comment is associated with.
        user_id INTEGER,                      -- ID of the user who made the comment.
        body TEXT NOT NULL,                   -- Content of the comment.
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp of when the comment was created.
        FOREIGN KEY (post_id) REFERENCES posts(id), -- Relationship to the "posts" table.
        FOREIGN KEY (user_id) REFERENCES users(id) -- Relationship to the "users" table.
    );`

	// SQL query to create the `likes_dislikes` table if it does not already exist.
	createLikesTable := `
	CREATE TABLE IF NOT EXISTS likes_dislikes (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for the like/dislike.
		user_id INTEGER,                      -- ID of the user giving the reaction.
		target_id INTEGER,                    -- ID of the post or comment being reacted to.
		target_type TEXT,                     -- Type of the target: 'post' or 'comment'.
		is_like BOOLEAN,                      -- TRUE for like, FALSE for dislike.
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp of when the reaction was made.
		FOREIGN KEY (user_id) REFERENCES users(id), -- Relationship to the "users" table.
		UNIQUE (user_id, target_id, target_type) -- Ensure each user can react to a target only once.
	);`

	// SQL query to create the `sessions` table if it does not already exist.
	createSessionsTable := `
    CREATE TABLE IF NOT EXISTS sessions (
		user_id INTEGER,                      -- ID of the user associated with the session.
		email TEXT UNIQUE,                    -- Email of the user for session validation.
		session_token TEXT NOT NULL,          -- Unique token for maintaining session state.
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP -- Timestamp of when the session was created.
    );`

	// Execute each SQL query and handle potential errors.
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

	// Return nil to indicate success if no errors occurred.
	return nil
}

// addMockData populates the database with sample data if it's empty.
func addMockData(db *sql.DB) {
	// Check if there are any existing users in the `users` table.
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		// Log an error message if the query fails, but do not terminate the application.
		log.Println("Error checking user count:", err)
		return
	}

	if count == 0 { // Check if the 'count' variable indicates that there are no users in the database.
		log.Println("No users found, inserting mock data...") // Log a message to indicate that mock data will be inserted.

		// Insert mock users
		_, err = db.Exec(`INSERT INTO users (username, password_hash, email) VALUES
        ('alice', 'fakehashedpassword1', 'a@q'),
        ('bob', 'fakehashedpassword2', 'a@h'),
        ('carol', 'fakehashedpassword3', 'a@j');`)
		// Execute a SQL query to insert three mock users with predefined usernames, hashed passwords, and email addresses.

		if err != nil { // Check if an error occurred during the execution of the query.
			log.Println("Error inserting mock users:", err) // Log the error message if insertion fails.
			return                                          // Exit the function to prevent further execution if an error occurs.
		}

		// Inserting mock categories
		_, err = db.Exec(`INSERT INTO categories (id, name, description) VALUES
		 (1, 'General Discussion', 'All about daily conversations and socializing.'),
		 (2, 'Books & Reviews', 'Book discussions, reviews, and recommendations.'),
		 (3, 'Literary Analysis', 'Deep dives into themes, symbols, and meanings in literature.'),
		 (4, 'Writing & Creativity', 'Tips, exercises, and discussions for writers and creative minds.'),
		 (5, 'Author Spotlight', 'Exploring the lives and works of notable authors.');`)
		// Execute a SQL query to insert mock categories with unique IDs, names, and descriptions.

		if err != nil { // Check if an error occurred during the execution of the query.
			log.Println("Error inserting mock categories:", err) // Log the error message if insertion fails.
			return                                               // Exit the function to prevent further execution if an error occurs.
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
		// Execute a SQL query to insert mock posts with specified IDs, associated user IDs, titles, content, and category IDs.

		if err != nil { // Check if an error occurred during the execution of the query.
			log.Println("Error inserting mock posts:", err) // Log the error message if insertion fails.
			return                                          // Exit the function to prevent further execution if an error occurs.
		}

		// Insert mock comments
		_, err = db.Exec(`INSERT INTO comments (post_id, user_id, body) VALUES
		 (1, 2, 'Nice post, Alice!'),
		 (2, 1, 'Thanks, Bob! Great thoughts!'),
		 (3, 2, 'I completely agree with Carol''s points.');`)
		// Execute a SQL query to insert mock comments associated with specific posts and users.

		if err != nil { // Check if an error occurred during the execution of the query.
			log.Println("Error inserting mock comments:", err) // Log the error message if insertion fails.
			return                                             // Exit the function to prevent further execution if an error occurs.
		}

		log.Println("Mock data inserted successfully.") // Log a message to indicate that all mock data has been inserted successfully.
	}
}
