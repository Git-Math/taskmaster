/* ************************************************************************** */
/*                                                                            */
/*                                                        :::      ::::::::   */
/*   loop-print-stdout.c                                :+:      :+:    :+:   */
/*                                                    +:+ +:+         +:+     */
/*   By:  <>                                        +#+  +:+       +#+        */
/*                                                +#+#+#+#+#+   +#+           */
/*   Created: 2020/07/14 15:58:56 by                   #+#    #+#             */
/*   Updated: 2020/09/23 23:46:32 by                  ###   ########.fr       */
/*                                                                            */
/* ************************************************************************** */

#include <stdio.h>
#include <unistd.h>
#include <signal.h>

static void intercept_signal(int sig)
{
	fprintf(stdout, "Gotcha! %d", sig);
}

int main()
{
	signal(SIGINT, &intercept_signal);
	while (1) {
		fprintf(stdout, "Just waiting for my signal..\n");
		sleep(2);
	}
	return 0;
}
