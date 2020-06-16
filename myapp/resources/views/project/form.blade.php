<div class="container">
    <div class="row">
        <div class="col-md-12">
            @include('project.message')
            <div class="card">
                <div class="card-header">{{ __('messages.project.create') }}</div>
                <div class="card-body">
                    @if($target == 'store')
                    <form action="/project" method="post">
                    @elseif($target == 'update')
                    <form action="/project/{{ $project->project_no }}" method="post">
                        <input type="hidden" name="_method" value="PUT">
                    @endif
                        @csrf
                        <div class="form-group">
                            <label for="project_no">{{ __('messages.project.no') }}</label>
                            @if ($target == 'store')
                            <input type="text" class="form-control" name="project_no" value="{{ $project->project_no }}">
                            @else
                            <input type="text" class="form-control" name="project_no" value="{{ $project->project_no }}" readonly>
                            @endif
                        </div>
                        <button type="submit" class="btn btn-outline-primary">{{ __('messages.register') }}</button>
                        <a href="/project">{{ __('messages.back') }}</a>
                    </form>
                </div>
            </div>
        </div>
    </div>
</div>
