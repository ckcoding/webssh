import { getLanguage } from '@/lang/index'

export default {
    sshInfo: {
        host: '',
        ip: '',
        countryFlag: '',
        location: '',
        os: '',
        memory: '',
        expiresAt: '',
        username: 'root',
        port: 22,
        password: '',
        logintype: 0
    },
    sshList: Object.prototype.hasOwnProperty.call(localStorage, 'sshList') ? localStorage.getItem('sshList') : null,
    savePass: true,
    savedSSHList: [],
    termList: [],
    currentTab: {},
    systemInfo: {},
    language: getLanguage()
}
