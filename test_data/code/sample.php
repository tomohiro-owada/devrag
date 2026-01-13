<?php

namespace App\Services;

use App\Repository\UserRepository;
use App\Models\User;
use Illuminate\Support\Facades\Log;

/**
 * UserService handles user-related business logic
 */
class UserService
{
    private UserRepository $userRepository;
    private CacheService $cacheService;

    public function __construct(UserRepository $userRepository, CacheService $cacheService)
    {
        $this->userRepository = $userRepository;
        $this->cacheService = $cacheService;
    }

    /**
     * Find a user by their ID
     */
    public function findById(int $id): ?User
    {
        $cacheKey = "user:{$id}";

        $cached = $this->cacheService->get($cacheKey);
        if ($cached !== null) {
            return $cached;
        }

        $user = $this->userRepository->find($id);
        if ($user !== null) {
            $this->cacheService->set($cacheKey, $user, 3600);
        }

        return $user;
    }

    /**
     * Create a new user
     */
    public function createUser(array $data): User
    {
        Log::info('Creating user', ['email' => $data['email']]);

        $user = new User();
        $user->fill($data);
        $this->userRepository->save($user);

        return $user;
    }

    /**
     * Update user profile
     */
    public function updateProfile(User $user, array $data): User
    {
        $user->fill($data);
        $this->userRepository->save($user);
        $this->cacheService->delete("user:{$user->id}");

        return $user;
    }
}

interface CacheService
{
    public function get(string $key): mixed;
    public function set(string $key, mixed $value, int $ttl): void;
    public function delete(string $key): void;
}

trait Cacheable
{
    protected array $cacheKeys = [];

    public function getCacheKey(): string
    {
        return static::class . ':' . $this->id;
    }

    public function clearCache(): void
    {
        foreach ($this->cacheKeys as $key) {
            cache()->forget($key);
        }
    }
}

function calculateDiscount(float $price, float $percentage): float
{
    return $price * (1 - $percentage / 100);
}
