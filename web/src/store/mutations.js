export default {
    SET_PASS(state, pass) {
        state.sshInfo.password = pass
    },
    SET_SAVE_PASS(state, savePass) {
        state.savePass = Boolean(savePass)
    },
    SET_LIST(state, list) {
        state.sshList = list
        localStorage.setItem('sshList', list)
    },
    SET_SAVED_LIST(state, list) {
        state.savedSSHList = list
    },
    SET_TERMLIST(state, list) {
        state.termList = list
    },
    SET_SSH(state, ssh) {
        state.sshInfo.host = ssh.host || ''
        state.sshInfo.ip = ssh.ip || ''
        state.sshInfo.countryFlag = ssh.countryFlag || ''
        state.sshInfo.location = ssh.location || ''
        state.sshInfo.os = ssh.os || ''
        state.sshInfo.memory = ssh.memory || ''
        state.sshInfo.expiresAt = ssh.expiresAt || ''
        state.sshInfo.username = ssh.username || 'root'
        state.sshInfo.port = Number(ssh.port) || 22
        state.sshInfo.logintype = ssh.logintype === undefined ? 0 : Number(ssh.logintype)
        state.sshInfo.password = ssh.password === undefined ? '' : ssh.password
    },
    SET_TAB(state, tab) {
        state.currentTab = tab
    },
    SET_SYSTEM_INFO(state, info) {
        state.systemInfo = info || {}
    },
    CLEAR_SYSTEM_INFO(state) {
        state.systemInfo = {}
    },
    SET_LANGUAGE: (state, language) => {
        state.language = language
        localStorage.setItem('language', language)
    }
}
