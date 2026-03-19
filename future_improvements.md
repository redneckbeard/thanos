# Future Improvements

## Clearer error messages for unresolved methods

When a gem method's body references a method that doesn't exist in the type system
(e.g., `a.dup()`), the error surfaces as a compiler panic:

    Method not set on MethodCall (a.dup())

This is confusing. The actual problem is "Array has no method 'dup'" — which was
detected during analysis but swallowed by tolerant mode. The compiler then hits a
nil Method field and panics.

We should either:
- Emit the original "no known method" error as a warning when tolerant mode swallows it
- Or have the compiler produce a clear message like "unresolved method 'dup' on Array(IntType)"
  instead of panicking with an internal-sounding "Method not set" message
