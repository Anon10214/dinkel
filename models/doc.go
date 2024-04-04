/*
Package models holds all fuzzing targets.

Each fuzzing target has an implementation struct, implementing [translator.Implementation]
as well as a driver struct implementing [dbms.DB] .
*/
package models

/*
The code below serves no functionality other than allowing the docstring of this package
to correctly link to translator.Implementation and dbms.DB.
*/

import (
	"fmt"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/translator"
)

func _() {
	fmt.Print(dbms.Valid, translator.DropIns(nil))
}
