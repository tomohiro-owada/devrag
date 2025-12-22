class User:
    def __init__(self, user_id, name, email):
        self.id = user_id
        self.name = name
        self.email = email


class UserService:
    def __init__(self):
        self.users = {}

    def get_user(self, user_id):
        return self.users.get(user_id)

    def create_user(self, name, email):
        user_id = len(self.users) + 1
        user = User(user_id, name, email)
        self.users[user_id] = user
        return user


def greet(name):
    return f"Hello, {name}!"
