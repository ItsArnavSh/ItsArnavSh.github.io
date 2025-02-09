``
## Introduction
# 🤔 How Do We Run This File?

Imagine we're handed a file with the following code:
```javascript
let a = 2;
let b = a * 2;
console.log("The Solution is ", b);
```
**How on earth are we supposed to make sense of this string of text?!** 🤯
That's where **an interpreter** swoops in to save the day! 🎉

## 🌟 The Job of an Interpreter

The interpreter's main task is to turn this **cryptic text** into something meaningful that a computer can execute. 🖥️✨ How does it do that? By following **3 steps**.

### Step 1️⃣: **Scanning and Tokenizing**

The first thing we need to do is **scan** the text and break it into **tokens**. 🔍 A token is the smallest meaningful piece of code. Think of it like the building blocks of the program:

- **Keywords** (e.g., `let`, `function`)
- **Literals** (e.g., numbers like `2`, strings like `"hello"`)
- **Identifiers** (e.g., variable names like `a`, `b`)
- **Operators** (e.g., `=`, `*`)

In any programming language, **tokens** usually have:

1. A **type** (e.g., `IDENTIFIER`, `NUMBER_LITERAL`)
2. A **value**, if it holds one (e.g., `2` is a `NUMBER_LITERAL` with a value of `2`).

For example:
```
let -> LET_KEYWORD
a   -> IDENTIFIER   "a"
=   -> EQUAL_SIGN   (no value, type says it all)
2   -> NUMBER_LITERAL 2
...etc

```
This process of scanning and tokenizing is done by the **Scanner** (or Lexer), which transforms the raw code into a list of meaningful tokens. 🪄✨ Here's what the scanner does:
```
Code: `let a = 2;`
Tokens:
1. { type: LET_KEYWORD }
2. { type: IDENTIFIER, value: "a" }
3. { type: EQUAL_SIGN }
4. { type: NUMBER_LITERAL, value: 2 }

```
## Step 2️⃣: Parsing The Text
Now that we have the list of tokens with us, we need to parse it. Parsing is all about making sense of it, and converting it into an Abstract Syntax Tree.

Before all of that, here is a quick recap on how trees work in programming. 🌳✨

---

### 🌲 What is a Tree in Programming?

A **tree** is a special type of data structure that resembles a hierarchy. Think of it like a family tree or the folder structure on your computer. 🗂️ Here's how it works:

- The topmost element is called the **root**. 🌟
- Every item (called a **node**) can have **children** nodes branching out below it.
- Nodes with no children are called **leaves**. 🍃
- The connections between nodes are called **edges**.

---

### 🛠️ Key Features of Trees

1. **Parent-Child Relationship:**
   Each node is connected to its parent and may have zero or more children. For example:

   ```plaintext
         Root
         /  \
      Node1  Node2
       /       \
    Leaf1     Leaf2
```
```
2. **Single Path Rule:**
There is only one path between any two nodes, meaning no cycles! It's not a tree if you can loop back to the root. 🔄❌

**Now with all that logic, lets see how that will be used to build an abstract syntax tree**

Quick which of the two equations make sense?
```
let a = 2+3;
let = a + 2 3;
```
The second one right?
Its because the second one follows a set of rules, a grammar. Without a proper grammar, we all would be speaking and writing gibberish and both us and computers would have a hard time deciphering what the message actually means.

In CS, before making the parser, we first construct the Grammar of the language we will be working on, so that we know we are on the right track.  A formal grammar definition though sounds intimidating like one of those Math Derivations, but trust me, it really is pretty trivial.

## Grammar 📚

There are two basic terms in Parsing Grammar: **Terminals** and **Non-Terminals**. If you imagine a tree analogy, **terminals** are essentially the leaf nodes, while **non-terminals** are all the other nodes. 🌳

### Example 1

Let’s illustrate this with an example:

**Consider:**
`a = b + c`

This is converted into a tree branch that looks like this:

```plaintext
           =
         /  \
         a    +
             / \
            b   c
```

(_Notice I’m calling it a branch because it is part of a much larger tree representing the whole program. This is just a single statement within that tree._)

Makes sense? We’re breaking it down into a meaningful structure!

### Explanation 🧩

- **Non-terminals**: `=`, `+`
    - If these are present, it implies there should be something to the left, right, or both for the given statement to be grammatically correct.
- **Terminals**: `a`, `b`, `c`
    - These are the leaf nodes and do not enforce any further structure.

### Example 2

Let’s enhance this logic with another example:

**Consider:**
`a = b + c * d`

This converts into the following branch of the tree:

```plaintext
           =
         /  \
         a    +
             / \
            b   *
               /  \
              c    d
```

Now it’s even clearer:

- **Terminals**: `a`, `b`, `c`, `d`
- **Non-terminals**: `=`, `+`, `*`

### Summary ✨

Operators (`=`, `+`, `*`) are **non-terminals**, as they structure the statement. Variables (`a`, `b`, `c`, `d`) are **terminals**, as they represent the final values.

Great! Now, how do we actually define the rules we need to play with? 🤔


## Syntax for Grammar ✍️

There is a popular formal syntax for grammar called **BNF**. You can look it up—Computerphile has some amazing videos on it! However, I’ll discuss a much simpler regex-like syntax as explained in the book _Crafting Interpreters_.

I don’t want to overwhelm you with learning the entire thing, as there isn’t much to it. Instead, let’s construct our grammar for a language side by side.

### Dummy Language Example 💡

Here’s a dummy language whose syntax we will build:

```plaintext
void name{
a = 2 + 3;
b = 10 + 20;
}
```

This language has no arguments in functions for simplicity, avoiding messy string handling for now. Let’s keep things clean! 🧹

---

### Building the Grammar 🛠️

Let’s break down the **hierarchy of components** in our code and build the structure step by step.

#### What is a program made of? 🤔

A program is made of **functions**, which we can write in our syntax as:

```plaintext
Program = Function+
```

- `+` means **1 or more**.
- Similarly, `*` means **0 or more**.

Since a program needs at least one function, we use `+`.

#### What is a function?

```plaintext
Function = header '{' body '}'
```

Here:

- **`header`**: `void name`
- **`body`**: The statements inside the curly braces.

#### Defining Header and Body

```plaintext
header = DataType Identifier
body = Statement+
```

- **`header`**: Contains a **DataType** and an **Identifier** in this order.
- **`body`**: Contains **1 or more statements**.

#### Defining DataType and Identifier

```plaintext
DataType = int | float | char | void
Identifier = (alphabet | _ ) (alphanumeric | _ )*
```

- `|` means "or".
- A **DataType** can be any of the listed types.
- An **Identifier** starts with an **alphabet** or an **underscore**, followed by any number of **alphanumeric** or **underscore** characters.

The `*` here allows **0 or more characters** after the first one.

---

### Diagnosing Statements 🧐

```plaintext
Statement = identifier '=' expression;
```

#### Defining an Expression

For simplicity, let’s say an expression consists of **numbers**, **identifiers**, and operators (`+`, `*`):

```plaintext
expression = terminal | (expression operator expression)
operator = + | *
terminal = identifier | number
```

This definition is simple and elegant but lacks **operator precedence** rules, which are critical for **order of operations (BODMAS)**.

---

### Addressing Operator Precedence ⚖️

#### Issue:

Given an expression like:

```plaintext
2 + 3 * 4
```

Our current grammar might parse it as:

```plaintext
(2 + 3) * 4
```

Instead of the correct BODMAS evaluation:

```plaintext
2 + (3 * 4)
```

#### Solution:

To fix this, we adjust the grammar to handle precedence by **splitting it into levels**:

```plaintext
expression = term ( '+' term )*
term = factor ( '*' factor )*
factor = identifier | number | '(' expression ')'
```

#### Explanation:

- **`term`** resolves multiplication first.
- **`expression`** handles addition after terms are resolved.
- **`factor`** includes parentheses to allow manual precedence overriding.

With this grammar, the expression `2 + 3 * 4` is correctly parsed as:

```plaintext
2 + (3 * 4)
```

### How Does This Work?

- **`term`** handles multiplication first because it’s defined at a lower level.
- **`expression`** handles addition only after terms are resolved.
- **`factor`** includes parentheses to allow overriding precedence manually.

Using this grammar, the expression `2 + 3 * 4` is parsed as:

```
expression -> term '+' term
term -> factor '*' factor
factor -> 3, 4
```

This ensures multiplication (`3 * 4`) happens first, followed by addition (`2 + 12`), yielding the correct result `14`.

## The parse tree in Action

### Grammar till now:

```
Program = Function+
Function = Header '{' Body '}'
Header = DataType Identifier
Body = Statement+
Statement = Identifier '=' Expression
Expression = Terminal | (Expression Operator Expression)
Operator = '+' | '*'
Terminal = Identifier | Number
DataType = 'int' | 'float' | 'char' | 'void'
Identifier = (alphabet | _ ) (alphanumeric | _ )*
```

### Example Code:

```c
void func1 {
    a = b + c * d;
}
void func2 {
    x = y + z;
}
```

---

### Parse Tree (ASCII Style)

```
Program
├── Function (func1)
│   ├── Header
│   │   ├── DataType (void)
│   │   └── Identifier (func1)
│   ├── Body
│   │   └── Statement
│   │       ├── Identifier (a)
│   │       ├── '='
│   │       └── Expression
│   │           ├── Identifier (b)
│   │           ├── '+'
│   │           └── Expression
│   │               ├── Identifier (c)
│   │               ├── '*'
│   │               └── Identifier (d)
├── Function (func2)
│   ├── Header
│   │   ├── DataType (void)
│   │   └── Identifier (func2)
│   ├── Body
│   │   └── Statement
│   │       ├── Identifier (x)
│   │       ├── '='
│   │       └── Expression
│   │           ├── Identifier (y)
│   │           ├── '+'
│   │           └── Identifier (z)
```

---

### Parse Tree (Lateral Style)

#### For Function `func1`:

```
               =
             /   \
            a     +
                 / \
                b   *
                   / \
                  c   d
```

#### For Function `func2`:

```
               =
             /   \
            x     +
                 / \
                y   z
```

### Full Program Tree:

```
Program
   ├── Function (func1)
   │         \
   │          =
   │         / \
   │        a   +
   │           / \
   │          b   *
   │             / \
   │            c   d
   └── Function (func2)
             \
              =
             / \
            x   +
               / \
              y   z
```

This breakdown shows how the program can be parsed into a tree format for easier interpretation.

## Sneak Peek into the implementation
I would encourage you to go ahead and try writing a small piece of code, maybe just a calculator(Thats what I did first) to experiment with making a parse tree. A very good explanation and a highly detailed on is given in Crafting Interpreters, and you should check that out, for constructing the parse tree. However, here is a look into how thats done

So we have a counter that keeps track of the current token's index, it will start at 0 and go all the way till the length of the list of tokens. Now we have two functions, match and consume. match(token) will take a token and check if the current token is same as the one sent in it, and return a boolean.
consume(token,errorMsg) will take the type of token expected there and an error message. If the token is not the one we sent, it will crash the code with the given error message. Again, for implementation details check the book, its a goldmine. I am here just to give you the jist of it.

## The Walker and Post Order Traversal
Now, we have constructed a complete tree, and have to go ahead and parse the tree gracefully, executing it. For that, we will start at the root, and do a post order traversal.
Here is some theory, and then I will give an example with experssion

### Post-Order Traversal (L-R-Root)

In **post-order traversal**, you visit:

1. **Left subtree**.
2. **Right subtree**.
3. **Root node**.

#### Example:

For a binary tree:

```
      A
     / \
    B   C
   / \
  D   E
```

#### Steps:

1. Visit the **left subtree** (`B → D → E`).
2. Visit the **right subtree** (`C`).
3. Visit the **root** (`A`).

#### Post-Order Traversal Output:

```
D → E → B → C → A
```

**Key Rule**: Process children first, then the parent.


---

### Example:

Consider the expression represented by the following binary tree:

```
               =
             /   \
            a     +
                 / \
                b   *
                   / \
                  c   d
```

Where:

- `b = 1`
- `c = 2`
- `d = 3`

### Post-Order Traversal Approach:

1. **Start at the root**: The root is the `=` sign, representing an assignment. To perform this assignment, we need to first evaluate both the **left-hand side (LHS)** and the **right-hand side (RHS)** of the equation. This requires us to traverse both sides of the tree using **post-order traversal**.

2. **Left Subtree (LHS)**: The left side is simply `a`, a terminal identifier. No further traversal is needed here, so we just take `a` as is.

3. **Right Subtree (RHS)**: The right-hand side is the `+` operator, which is a non-terminal. So, we need to perform post-order traversal on the right subtree, which consists of:

    - The left child: `b` (terminal with value `1`).
    - The right child: `*` (another non-terminal).
4. **Subtree with `*` (Right of `+`)**: The multiplication operator (`*`) requires post-order traversal as well. It has two children:

    - `c` (terminal with value `2`).
    - `d` (terminal with value `3`).

    Since both `c` and `d` are terminals, we evaluate them first: `c = 2` and `d = 3`. The multiplication (`c * d`) gives us `2 * 3 = 6`. Now the `*` node becomes:

    ```
                *
              /   \
             2     3
    ```

    We replace the `*` node with `6`:

    ```
                +
              /   \
             b     6
    ```

5. **Final Step**: Now that we have resolved both children of the `+` node (`b = 1` and the result of `* = 6`), we can evaluate the addition: `1 + 6 = 7`. So, the `+` node is replaced with `7`:

    ```
                =
              /   \
             a     7
    ```

6. **Solve the Assignment**: Finally, we can assign the value of `7` to `a`:


This is the final result of the post-order traversal and evaluation process for this simple expression.

### Key Points:

- **Post-order Traversal**: This process visits the left subtree first, then the right subtree, and finally the root, ensuring that we evaluate operands before operators.
- **Recursive Traversal**: Non-terminals (like `+` and `*`) require recursive traversal to break down the expression into simpler components, ultimately reducing the expression to terminal values that can be computed.
- **Evaluation**: Once the tree has been fully traversed and reduced to terminal values, the final result can be computed step by step.

### Extending This Concept:

This approach can be extended to handle more complex operations such as strings, loops, conditionals (if-else), and even multi-level expressions involving arrays or objects. It is a foundational concept for building parsers and evaluators for your own programming language.

### Project Suggestion:

A great beginner project would be to build a **calculator** that can handle multiple arithmetic operations:

- For example, you could write an expression like:

    ```
    a = 2 + 4
    b = a / 2 + 20
    ```

    And print the results of the expressions.

Once you've implemented that, you can explore **scope management** (keeping track of variable values) and **control flow** (like `if` statements, loops), ultimately creating a simple **interpreter or mini-language**.

---

## Limitations
Tree walk interpreters are simple and easy to implement but suffer from poor performance due to direct evaluation of the Abstract Syntax Tree (AST) at runtime, without optimizations like operator precedence or memory management. They are slower than compiled languages because they lack optimizations and perform redundant work during interpretation. In contrast, machine code compilation converts code directly to machine language, offering excellent performance but no flexibility. Bytecode compilation translates code into a platform-independent format, offering portability and some optimizations, while Just-In-Time (JIT) compilation dynamically compiles code at runtime, providing performance close to machine code with the advantage of dynamic optimizations, though it introduces initial overhead. I insist you to check these other compilation methods out and try stepping up your model later on.
