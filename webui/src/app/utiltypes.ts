// Type that allows easily override nested interfaces
// Source: https://stackoverflow.com/a/65561287
export type ModifyDeep<A extends AnyObject, B extends DeepPartialAny<A>> = {
    // For each key K in A: Check if B has a K key too.
    [K in keyof A]: K extends keyof B
        ? // B has the K key: Check if B[K] is an array.
          B[K] extends Array<infer C>
            ? // B[K] is an array: Check if A[K] is an array too.
              A[K] extends Array<infer D>
                ? // A[K] and B[K] are the arrays: Modify the array entry type.
                  Array<ModifyDeep<D, C>>
                : // B[K] is an array but A[K] no: Use B[K].
                  B[K]
            : // A[K] and B[K] are not the arrays: Check if B[K] is an object.
            B[K] extends AnyObject
            ? // B[K] is an object: Modify the A[K] using B[K].
              ModifyDeep<A[K], B[K]>
            : // A[K] and B[K] are not the objects: Use B[K].
              B[K]
        : // B doesn't have the K key: Use A[K].
          A[K]
} & (A extends AnyObject ? Omit<B, keyof A> : A)

// Makes each property optional and turns each leaf property into any,
// allowing for type overrides by narrowing any. */
type DeepPartialAny<T> = {
    [P in keyof T]?: T[P] extends AnyObject ? DeepPartialAny<T[P]> : any
}

type AnyObject = Record<string, any>
