import { FormControl, FormGroup } from "@angular/forms"
import { PasswordPolicy } from "./password-policy"


describe('PasswordPolicy', () => {
    it('should verify if the passwords are the same', () => {
        const formGroup = new FormGroup({
            oldPassword: new FormControl('password'),
            newPassword: new FormControl('password'),
        })

        const validator = PasswordPolicy.differentPasswords('oldPassword', 'newPassword')
        expect(validator(formGroup)).toEqual({ samePasswords: true })
    })

    it('should verify if the passwords are not the same', () => {
        const formGroup = new FormGroup({
            oldPassword: new FormControl('password'),
            newPassword: new FormControl('another-password'),
        })

        const validator = PasswordPolicy.differentPasswords('oldPassword', 'newPassword')
        expect(validator(formGroup)).toBeNull()
    })
})