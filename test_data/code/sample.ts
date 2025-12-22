interface User {
  id: number;
  name: string;
  email: string;
}

class UserService {
  private users: Map<number, User> = new Map();

  getUser(id: number): User | undefined {
    return this.users.get(id);
  }

  createUser(name: string, email: string): User {
    const id = this.users.size + 1;
    const user: User = { id, name, email };
    this.users.set(id, user);
    return user;
  }
}

function formatUserName(user: User): string {
  return `${user.name} <${user.email}>`;
}

const fetchUser = async (id: number): Promise<User> => {
  return { id, name: "Test", email: "test@example.com" };
};
