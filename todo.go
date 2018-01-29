package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	"github.com/mkideal/cli"
)

type todo struct {
	ID   int
	Text string
}

// root command
type rootT struct {
	cli.Helper
	Name string `cli:"name" usage:"your name"`
}

// get command
type getT struct {
	cli.Helper
	ID int `cli:"id" usage:"todo item id"`
}

// add command
type addT struct {
	cli.Helper
	Task string `cli:"task" usage:"todo item text"`
}

// remove command
type removeT struct {
	cli.Helper
	ID int `cli:"id" usage:"remove a todo item"`
}

// edit command
type editT struct {
	cli.Helper
	ID   int    `cli:"id" usage:"todo id"`
	Task string `cli:"task" usage:"updated text"`
}

// list command
type listT struct {
	cli.Helper
}

func main() {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("todos"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	defer db.Close()

	var help = cli.HelpCommand("display help information")

	var root = &cli.Command{
		Desc: "this is root command",
		// Argv is a factory function of argument object
		// ctx.Argv() is if Command.Argv == nil or Command.Argv() is nil
		Argv: func() interface{} { return new(rootT) },
		Fn: func(ctx *cli.Context) error {
			argv := ctx.Argv().(*rootT)
			ctx.String("Hello, root command, I am %s\n", argv.Name)
			return nil
		},
	}

	var get = &cli.Command{
		Name: "get",
		Desc: "get item",
		Argv: func() interface{} { return new(getT) },
		Fn: func(ctx *cli.Context) error {
			argv := ctx.Argv().(*getT)
			var id = argv.ID
			db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("todos"))
				value := b.Get(itob(id))
				var todoSamp todo
				err := json.Unmarshal(value, &todoSamp)
				if err != nil {
					fmt.Println("error:", err)
				}
				fmt.Printf("%d)  %s\n", todoSamp.ID, todoSamp.Text)
				return nil
			})
			return nil
		},
	}

	var add = &cli.Command{
		Name: "add",
		Desc: "add item",
		Argv: func() interface{} { return new(addT) },
		Fn: func(ctx *cli.Context) error {
			argv := ctx.Argv().(*addT)
			var text = argv.Task
			err := db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("todos"))

				// Generate ID for the user.
				id, _ := b.NextSequence()
				newTodo := todo{int(id), text}
				// Marshal todo data into bytes.
				buf, err := json.Marshal(newTodo)
				if err != nil {
					return err
				}

				ctx.String("%d) %s\n", newTodo.ID, newTodo.Text)
				// Persist bytes to users bucket.
				return b.Put(itob(newTodo.ID), buf)
			})
			if err == nil {
				ctx.String("Todo succesfully added")
			} else {
				fmt.Println("Error adding todo:", err)
			}
			return nil
		},
	}

	var remove = &cli.Command{
		Name: "remove",
		Desc: "remove item",
		Argv: func() interface{} { return new(removeT) },
		Fn: func(ctx *cli.Context) error {
			argv := ctx.Argv().(*removeT)
			var id = argv.ID
			// Remove the item from the db.
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("todos"))
				err := b.Delete(itob(id))
				if err == nil {
					ctx.String("Removed, todo - %d\n", id)
				} else {
					fmt.Println("Error deleting item:", err)
				}
				return nil
			})

			return nil
		},
	}

	var edit = &cli.Command{
		Name: "edit",
		Desc: "edit item",
		Argv: func() interface{} { return new(editT) },
		Fn: func(ctx *cli.Context) error {
			argv := ctx.Argv().(*editT)
			var text = argv.Task
			var id = argv.ID
			update := todo{id, text}
			err := db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("todos"))

				// Marshal todo data into bytes.
				buf, err := json.Marshal(update)
				if err != nil {
					return err
				}

				// Persist bytes to users bucket.
				return b.Put(itob(update.ID), buf)
			})
			if err == nil {
				ctx.String("Todo succesfully edited")
			} else {
				fmt.Println("Error editing todo:", err)
			}
			return nil
		},
	}

	var list = &cli.Command{
		Name: "list",
		Desc: "list all items",
		Fn: func(ctx *cli.Context) error {
			ctx.String("Listing all items\n")
			db.View(func(tx *bolt.Tx) error {
				// Assume bucket exists and has keys
				b := tx.Bucket([]byte("todos"))

				c := b.Cursor()
				var todoSamp todo
				for k, v := c.First(); k != nil; k, v = c.Next() {
					err := json.Unmarshal(v, &todoSamp)
					if err != nil {
						fmt.Println("error:", err)
					}
					fmt.Printf("%d) %v\n", todoSamp.ID, todoSamp.Text)
				}
				return nil
			})
			ctx.String("End of list \n")
			return nil
		},
	}

	if err := cli.Root(root,
		cli.Tree(help),
		cli.Tree(add),
		cli.Tree(remove),
		cli.Tree(edit),
		cli.Tree(get),
		cli.Tree(list),
	).Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
