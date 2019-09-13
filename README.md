# gojvm

gojvm is an JVM implementation by Go.

It can interpret and run a JVM bytecode file.
Currently, it only supports "hello world" and arithmetic addition.

# USAGE

Hello world

```
$ cat HelloWorld.java                                                                                                                      (git)-[master] ? 
public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello world");
    }
}

$ cat HelloWorld.class | go run main.go                                                                                                 (git)-[master]  (p9)
Hello world
```

Arithmetic addition

```
$ cat Arith.java                                                                                                                           (git)-[master] ? 
public class Arith {
    public static void main(String[] args) {
        int c = sum(30, 12);
        System.out.println(c);
    }

    private static int sum(int a, int b) {
        return a + b;
    }
}

$ cat Arith.class | go run main.go                                                                                                         (git)-[master] ? 
42
```

# ACKNOWLEDGMENT

gjvm is inspired by [PHPJava](https://github.com/php-java/php-java).

I really appreciate the work.

# LICENSE

MIT

# AUTHOR

@DQNEO

