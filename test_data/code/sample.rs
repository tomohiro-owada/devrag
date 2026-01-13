use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::io::{self, Read, Write};

/// Configuration for the application
#[derive(Debug, Clone)]
pub struct Config {
    pub database_url: String,
    pub port: u16,
    pub debug: bool,
}

impl Config {
    /// Create a new configuration with defaults
    pub fn new() -> Self {
        Config {
            database_url: String::from("localhost:5432"),
            port: 8080,
            debug: false,
        }
    }

    /// Load configuration from environment variables
    pub fn from_env() -> Result<Self, ConfigError> {
        let database_url = std::env::var("DATABASE_URL")
            .map_err(|_| ConfigError::MissingEnvVar("DATABASE_URL"))?;

        let port = std::env::var("PORT")
            .unwrap_or_else(|_| "8080".to_string())
            .parse()
            .map_err(|_| ConfigError::InvalidPort)?;

        Ok(Config {
            database_url,
            port,
            debug: std::env::var("DEBUG").is_ok(),
        })
    }
}

impl Default for Config {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug)]
pub enum ConfigError {
    MissingEnvVar(&'static str),
    InvalidPort,
}

/// User entity
pub struct User {
    pub id: u64,
    pub name: String,
    pub email: String,
}

/// Repository for user operations
pub trait UserRepository {
    fn find(&self, id: u64) -> Option<User>;
    fn save(&mut self, user: User) -> Result<(), RepositoryError>;
    fn delete(&mut self, id: u64) -> Result<(), RepositoryError>;
}

#[derive(Debug)]
pub enum RepositoryError {
    NotFound,
    ConnectionError,
    DuplicateKey,
}

/// In-memory implementation of UserRepository
pub struct InMemoryUserRepository {
    users: Arc<Mutex<HashMap<u64, User>>>,
    next_id: u64,
}

impl InMemoryUserRepository {
    pub fn new() -> Self {
        InMemoryUserRepository {
            users: Arc::new(Mutex::new(HashMap::new())),
            next_id: 1,
        }
    }

    fn generate_id(&mut self) -> u64 {
        let id = self.next_id;
        self.next_id += 1;
        id
    }
}

impl UserRepository for InMemoryUserRepository {
    fn find(&self, id: u64) -> Option<User> {
        let users = self.users.lock().unwrap();
        users.get(&id).map(|u| User {
            id: u.id,
            name: u.name.clone(),
            email: u.email.clone(),
        })
    }

    fn save(&mut self, user: User) -> Result<(), RepositoryError> {
        let mut users = self.users.lock().unwrap();
        users.insert(user.id, user);
        Ok(())
    }

    fn delete(&mut self, id: u64) -> Result<(), RepositoryError> {
        let mut users = self.users.lock().unwrap();
        users.remove(&id).ok_or(RepositoryError::NotFound)?;
        Ok(())
    }
}

/// Service for user business logic
pub struct UserService<R: UserRepository> {
    repository: R,
}

impl<R: UserRepository> UserService<R> {
    pub fn new(repository: R) -> Self {
        UserService { repository }
    }

    pub fn get_user(&self, id: u64) -> Option<User> {
        self.repository.find(id)
    }

    pub fn create_user(&mut self, name: String, email: String) -> Result<User, RepositoryError> {
        let user = User {
            id: 0, // Will be assigned by repository
            name,
            email,
        };
        self.repository.save(user.clone())?;
        Ok(user)
    }
}

mod utils {
    /// Calculate hash for a string
    pub fn hash_string(s: &str) -> u64 {
        let mut hash: u64 = 5381;
        for c in s.bytes() {
            hash = ((hash << 5).wrapping_add(hash)).wrapping_add(c as u64);
        }
        hash
    }

    /// Format bytes as human-readable string
    pub fn format_bytes(bytes: u64) -> String {
        const KB: u64 = 1024;
        const MB: u64 = KB * 1024;
        const GB: u64 = MB * 1024;

        if bytes >= GB {
            format!("{:.2} GB", bytes as f64 / GB as f64)
        } else if bytes >= MB {
            format!("{:.2} MB", bytes as f64 / MB as f64)
        } else if bytes >= KB {
            format!("{:.2} KB", bytes as f64 / KB as f64)
        } else {
            format!("{} bytes", bytes)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_new() {
        let config = Config::new();
        assert_eq!(config.port, 8080);
        assert!(!config.debug);
    }

    #[test]
    fn test_user_repository() {
        let mut repo = InMemoryUserRepository::new();
        let user = User {
            id: 1,
            name: "Test".to_string(),
            email: "test@example.com".to_string(),
        };
        assert!(repo.save(user).is_ok());
        assert!(repo.find(1).is_some());
    }
}
