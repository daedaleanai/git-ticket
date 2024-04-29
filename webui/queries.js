const config = JSON.stringify({
    'bookmarks': {
        'Me': {
            'Assigned to me': 'assignee:javier',
            'Reported by me': 'author:javier'
        },
        'Milestones': {'PilotEye PDR': 'label:milestone:piloteye-pdr'},
        'All': {'All tickets': ''},
        'Impact': {
            'PLAT-137-SRD': 'label:impact:plat-srd',
            'VXS-121-DATA': 'label:impact:vxs-data'
        }
    }
})
localStorage.setItem('git-ticket', config)
