package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Apply writes the plan's changes into the project folder.
//
// Every file is checked before any is written, so a file that changed since
// the plan was shown stops Apply without leaving the project half updated.
func Apply(plan Plan) error {
	root, err := os.OpenRoot(plan.Path)
	if err != nil {
		return fmt.Errorf("cannot open the project folder: %w", err)
	}
	defer root.Close()

	for _, c := range plan.Changes {
		if err := checkUnchanged(root, c); err != nil {
			return err
		}
	}
	for _, c := range plan.Changes {
		if err := writeFileAtomically(root, c.Path, c.content); err != nil {
			return err
		}
	}
	return nil
}

// checkUnchanged makes sure the file is still as it was when the plan was
// built, and is not a symbolic link.
func checkUnchanged(root *os.Root, c Change) error {
	rerun := "Nothing was written. Run otelstarter init again to see an updated plan"

	info, err := root.Lstat(c.Path)
	if errors.Is(err, fs.ErrNotExist) {
		if c.isNew {
			return nil
		}
		return fmt.Errorf("%s was deleted after the plan was shown. %s", c.Path, rerun)
	}
	if err != nil {
		return fmt.Errorf("cannot check %s: %w", c.Path, err)
	}
	if c.isNew {
		return fmt.Errorf("%s was created after the plan was shown. %s", c.Path, rerun)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link, and otelstarter does not replace links. "+
			"Nothing was written. Add the changes shown above to the file by hand", c.Path)
	}

	current, err := fs.ReadFile(root.FS(), c.Path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", c.Path, err)
	}
	if !bytes.Equal(current, c.original) {
		return fmt.Errorf("%s changed after the plan was shown. %s", c.Path, rerun)
	}
	return nil
}

// writeFileAtomically writes to a temporary file and renames it over the
// target. A rename is all-or-nothing, so a crash or full disk never leaves a
// half-written file behind.
func writeFileAtomically(root *os.Root, name string, data []byte) error {
	perm := fs.FileMode(0o644)
	if info, err := root.Stat(name); err == nil {
		perm = info.Mode().Perm() // keep the existing file's permissions
	}

	tmp := name + ".otelstarter-tmp"
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("cannot write %s: %w", name, err)
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		root.Remove(tmp)
		return fmt.Errorf("cannot write %s: %w", name, err)
	}

	if err := root.Rename(tmp, name); err != nil {
		root.Remove(tmp)
		return fmt.Errorf("cannot write %s: %w", name, err)
	}
	return nil
}
