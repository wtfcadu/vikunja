import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import {projectOverviewQuery} from '@/client/queries/projectFinance'

export function useProjectOverview() {
	const query = useQuery(projectOverviewQuery())
	const items = computed(() => query.data.value ?? [])
	return {
		items,
		isPending: query.isPending,
		isFetching: query.isFetching,
	}
}
