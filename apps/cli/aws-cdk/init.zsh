# aws-cdk completions (yargs-based)
#_cdk_yargs_completions() {
#  local reply
#  local si=$IFS
#  IFS=$' '
#  reply=($(COMP_CWORD="$((CURRENT-1))" COMP_LINE="$BUFFER" COMP_POINT="$CURSOR" cdk --get-yargs-completions "${words[@]}"))
#  IFS=$si
#  _describe 'values' reply
#}
#compdef _cdk_yargs_completions cdk
