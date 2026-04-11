import { encodeSSHInfo } from '@/utils/ssh'

export default {
    sshReq: state => encodeSSHInfo(state.sshInfo)
}
