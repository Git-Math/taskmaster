/* ************************************************************************** */
/*                                                                            */
/*                                                        :::      ::::::::   */
/*   loop-write-file.c                                  :+:      :+:    :+:   */
/*                                                    +:+ +:+         +:+     */
/*   By:  <>                                        +#+  +:+       +#+        */
/*                                                +#+#+#+#+#+   +#+           */
/*   Created: 2020/09/27 17:19:10 by                   #+#    #+#             */
/*   Updated: 2020/09/27 18:15:45 by                  ###   ########.fr       */
/*                                                                            */
/* ************************************************************************** */

#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <signal.h>

int fd;

void sighandler(int sig)
{
	close(fd);
	exit(0);
}

int main()
{
	const char *f = getenv("WHERE");

	if (f == NULL) {
		fprintf(stderr, "error: env variable `WHERE` not set\n");
		return -1;
	}

	printf("info: writing to `%s`\n", f);

	fd = open(f, O_CREAT | O_WRONLY, 0777);
	if (fd < 0) {
		perror("open");
		return -1;
	}

	signal(SIGINT, &sighandler);

	for (int i = 0; i < 20; i++) {
		write(fd, "coucou la famille\n", sizeof("coucou la famille\n") - 1);
		sleep(2);
	}

	close(fd);
	return 0;
}
