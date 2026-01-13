class User:
    """Represents a user in the system."""

    def __init__(self, id: int, name: str, email: str):
        self.id = id
        self.name = name
        self.email = email


class UserService:
    """Provides user-related operations."""

    def __init__(self):
        self.users = {}

    def get_user(self, id: int):
        """Retrieves a user by ID."""
        return self.users.get(id)

    def create_user(self, name: str, email: str):
        """Creates a new user."""
        id = len(self.users) + 1
        user = User(id, name, email)
        self.users[id] = user
        return user


def greet(name: str) -> str:
    """Returns a greeting message."""
    return f"Hello, {name}!"


def calculate_sum(numbers: list) -> int:
    """Calculates the sum of numbers."""
    return sum(numbers)
