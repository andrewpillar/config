# Spec

This is the specification for the configuration language defined by this
library. The language was inspired by the NGINX configuration language,
primarily.

The full spec of the language is below in Extended Backus-Naur Form,

    digit  = "0" ... "9" .
    letter = "a" ... "z" | "A" ... "Z" | "_" | unicode_letter .

    identifier = letter { letter | digit } .

    float_literal  = digit "." { digit } .
    int_literal    = { digit } .
    number_literal = int_literal | float_literal .

    duration_unit    = "s" | "m" | "h" .
    duration_literal = number_literal { number_literal | duration_unit } .

    size_unit    = "B" | "KB" | "MB" | "GB" | "TB" .
    size_literal = int_literal size_unit .

    string_literal = `"` { letter } `"` .

    bool_literal = "true" | "false" .

    literal = bool_literal | string_literal | number_literal | duration_literal | size_literal .

    block   = "{" [ parameter ";" ] "}" .
    array   = "[" [ operand "," ] "]" .
    operand = literal | array | block .

    parameter = identifier [ identifier ] operand .

    file = { parameter ";" } .
