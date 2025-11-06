from bisect import bisect_left
import sys

def count_valid_subarrays(n, arr, queries):
    # previous index with value >= current
    left = [0]*(n+1)
    stack = []
    for i in range(1, n+1):
        val = arr[i-1]
        while stack and arr[stack[-1]-1] < val:
            stack.pop()
        left[i] = stack[-1] if stack else 0
        stack.append(i)

    # next index with value <= current
    right = [0]*(n+1)
    stack = []
    for i in range(n, 0, -1):
        val = arr[i-1]
        while stack and arr[stack[-1]-1] > val:
            stack.pop()
        right[i] = stack[-1] if stack else n+1
        stack.append(i)

    # sort indices by right bound to compute f
    indices = list(range(1, n+1))
    indices.sort(key=lambda x: right[x])

    f = [0]*(n+1)
    mx = 0
    p = 0
    for R in range(1, n+1):
        while p < n and right[indices[p]] <= R:
            mx = max(mx, left[indices[p]])
            p += 1
        f[R] = mx + 1

    A = [0]*(n+1)  # prefix of counts when start unrestricted
    F = [0]*(n+1)  # prefix sums of f
    for i in range(1, n+1):
        A[i] = A[i-1] + (i - f[i] + 1)
        F[i] = F[i-1] + f[i]

    res = []
    for l, r in queries:
        total = A[r] - A[l-1]
        # largest index in [l, r] with f[index] < l using binary search
        low, high = l, r
        best = l - 1
        while low <= high:
            mid = (low + high) // 2
            if f[mid] < l:
                best = mid
                low = mid + 1
            else:
                high = mid - 1
        t = best
        if t >= l:
            size = t - l + 1
            sumF = F[t] - F[l-1]
            extra = sumF - size * l
        else:
            extra = 0
        res.append(total + extra)
    return res

if __name__ == "__main__":
    input_data = sys.stdin.read().strip().split()
    it = iter(input_data)
    n = int(next(it))
    arr = [int(next(it)) for _ in range(n)]
    q = int(next(it))
    queries = [(int(next(it)), int(next(it))) for _ in range(q)]
    answers = count_valid_subarrays(n, arr, queries)
    print(' '.join(map(str, answers)))
