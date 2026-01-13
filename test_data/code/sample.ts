interface User {
  id: number;
  name: string;
  email: string;
}

interface UserRepository {
  getUser(id: number): User | undefined;
  createUser(name: string, email: string): User;
}

class UserService implements UserRepository {
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

  deleteUser(id: number): boolean {
    return this.users.delete(id);
  }
}

function greet(name: string): string {
  return `Hello, ${name}!`;
}

const calculateSum = (numbers: number[]): number => {
  return numbers.reduce((acc, n) => acc + n, 0);
};

export { User, UserRepository, UserService, greet, calculateSum };
